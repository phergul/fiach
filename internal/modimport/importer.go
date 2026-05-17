package modimport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phergul/mod-manager/internal/installconfig"
	"github.com/phergul/mod-manager/internal/storage"
)

type Source interface {
	Type() storage.ModSourceType
	OriginalPath() string
	OriginalName() *string
	SuggestedName() string
	Validate() error
	Materialize(destinationPath string) error
}

type Store interface {
	FindModByOriginalSourcePath(ctx context.Context, gameID int64, originalSourcePath string) (storage.Mod, bool, error)
	GetGlobalModStorageRoot(ctx context.Context) (string, error)
	ResolveGameModStoragePath(ctx context.Context, gameID int64, globalRoot string) (string, error)
	CreateOrReplaceModInstallConfig(ctx context.Context, input storage.CreateModInstallConfigInput) (storage.ModInstallConfig, error)
	CreateModWithInstallConfig(ctx context.Context, input storage.CreateModWithInstallConfigInput) (storage.CreateModWithInstallConfigResult, error)
	GetModInstallConfig(ctx context.Context, modID int64) (storage.ModInstallConfig, bool, error)
}

var unsafeManagedModFolderNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)
var repeatedManagedModFolderSeparators = regexp.MustCompile(`-+`)

func Import(ctx context.Context, store Store, gameID int64, name string, source Source, strategyType installconfig.StrategyType, targetRelativePath string) (result storage.CreateModWithInstallConfigResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("import mod source: %w", err)
		}
	}()

	if store == nil {
		return storage.CreateModWithInstallConfigResult{}, errors.New("store is not configured")
	}
	if source == nil {
		return storage.CreateModWithInstallConfigResult{}, errors.New("import source is required")
	}

	name, err = NormalizeName(name)
	if err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}
	if err := installconfig.ValidateSelectableStrategy(strategyType); err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}
	targetRelativePath, err = installconfig.NormalizeTargetRelativePath(targetRelativePath)
	if err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}

	existing, found, err := store.FindModByOriginalSourcePath(ctx, gameID, source.OriginalPath())
	if err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}
	configInput := storage.CreateModInstallConfigInput{
		StrategyType:       string(strategyType),
		TargetBase:         installconfig.TargetBaseGameRoot,
		TargetRelativePath: targetRelativePath,
	}
	if found {
		config, configFound, err := store.GetModInstallConfig(ctx, existing.ID)
		if err != nil {
			return storage.CreateModWithInstallConfigResult{}, err
		}
		if !configFound {
			configInput.ModID = existing.ID
			config, err = store.CreateOrReplaceModInstallConfig(ctx, configInput)
			if err != nil {
				return storage.CreateModWithInstallConfigResult{}, err
			}
		}
		return storage.CreateModWithInstallConfigResult{
			Mod:    existing,
			Config: config,
		}, nil
	}

	if err := source.Validate(); err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}

	globalRoot, err := store.GetGlobalModStorageRoot(ctx)
	if err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}

	gameStoragePath, err := store.ResolveGameModStoragePath(ctx, gameID, globalRoot)
	if err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}

	if err := os.MkdirAll(gameStoragePath, 0o755); err != nil {
		return storage.CreateModWithInstallConfigResult{}, fmt.Errorf("create game mod storage folder: %w", err)
	}
	if pathContains(gameStoragePath, source.OriginalPath()) {
		return storage.CreateModWithInstallConfigResult{}, fmt.Errorf("source %q contains the managed mod storage folder %q", source.OriginalPath(), gameStoragePath)
	}

	destinationPath, err := uniqueManagedModDestination(gameStoragePath, name)
	if err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}

	tempPath, err := makeImportTempDir(gameStoragePath, filepath.Base(destinationPath))
	if err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.RemoveAll(tempPath)
		}
	}()

	if err := source.Materialize(tempPath); err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}

	if _, err := os.Stat(destinationPath); err == nil {
		return storage.CreateModWithInstallConfigResult{}, fmt.Errorf("managed mod destination %q already exists", destinationPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return storage.CreateModWithInstallConfigResult{}, fmt.Errorf("check managed mod destination: %w", err)
	}

	if err := os.Rename(tempPath, destinationPath); err != nil {
		return storage.CreateModWithInstallConfigResult{}, fmt.Errorf("move managed mod folder into place: %w", err)
	}
	removeTemp = false

	removeDestination := true
	defer func() {
		if removeDestination {
			_ = os.RemoveAll(destinationPath)
		}
	}()

	result, err = store.CreateModWithInstallConfig(ctx, storage.CreateModWithInstallConfigInput{
		Mod: storage.CreateModInput{
			GameID:             gameID,
			Name:               name,
			SourceType:         source.Type(),
			SourcePath:         destinationPath,
			OriginalSourcePath: source.OriginalPath(),
			OriginalSourceName: source.OriginalName(),
		},
		Config: configInput,
	})
	if err != nil {
		return storage.CreateModWithInstallConfigResult{}, err
	}

	removeDestination = false
	return result, nil
}

func NormalizeName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("mod name is required")
	}

	return name, nil
}

func uniqueManagedModDestination(parent string, name string) (string, error) {
	baseName := managedModFolderName(name)
	for index := 0; ; index++ {
		candidateName := baseName
		if index > 0 {
			candidateName = fmt.Sprintf("%s-%d", baseName, index+1)
		}

		candidatePath := filepath.Join(parent, candidateName)
		_, err := os.Stat(candidatePath)
		if errors.Is(err, os.ErrNotExist) {
			return candidatePath, nil
		}
		if err != nil {
			return "", fmt.Errorf("check managed mod destination: %w", err)
		}
	}
}

func managedModFolderName(name string) string {
	name = strings.TrimSpace(name)
	name = unsafeManagedModFolderNameChars.ReplaceAllString(name, "-")
	name = repeatedManagedModFolderSeparators.ReplaceAllString(name, "-")
	name = strings.Trim(name, " .-")
	if name == "" {
		name = "mod"
	}

	return name
}

func pathContains(path string, potentialParent string) bool {
	path = filepath.Clean(path)
	potentialParent = filepath.Clean(potentialParent)
	if path == potentialParent {
		return true
	}

	relativePath, err := filepath.Rel(potentialParent, path)
	if err != nil {
		return false
	}

	return relativePath != "." && relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(os.PathSeparator))
}

func makeImportTempDir(parent string, destinationBaseName string) (string, error) {
	suffix, err := randomHexSuffix()
	if err != nil {
		return "", err
	}

	tempPath := filepath.Join(parent, "."+destinationBaseName+"-tmp-"+suffix)
	if err := os.Mkdir(tempPath, 0o755); err != nil {
		return "", fmt.Errorf("create temporary managed mod folder: %w", err)
	}

	return tempPath, nil
}

func randomHexSuffix() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate temporary folder suffix: %w", err)
	}

	return hex.EncodeToString(bytes), nil
}
