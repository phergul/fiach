package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/phergul/mod-manager/internal/storage"
)

var unsafeManagedModFolderNameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)
var repeatedManagedModFolderSeparators = regexp.MustCompile(`-+`)

func (s *ModService) ImportModFolder(gameID int64, name string, sourceFolderPath string) (mod storage.Mod, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("import mod folder: %w", err)
		}
	}()

	if s == nil || s.store == nil {
		return storage.Mod{}, errors.New("storage is not configured")
	}

	name, err = normalizeImportedModName(name)
	if err != nil {
		return storage.Mod{}, err
	}

	originalSourcePath, err := storage.CanonicalModOriginalSourcePath(sourceFolderPath)
	if err != nil {
		return storage.Mod{}, err
	}

	existing, found, err := s.store.FindModByOriginalSourcePath(context.Background(), gameID, originalSourcePath)
	if err != nil {
		return storage.Mod{}, err
	}
	if found {
		return existing, nil
	}

	if err := validateImportSourceFolder(originalSourcePath); err != nil {
		return storage.Mod{}, err
	}

	globalRoot, err := s.store.GetGlobalModStorageRoot(context.Background())
	if err != nil {
		return storage.Mod{}, err
	}

	gameStoragePath, err := s.store.ResolveGameModStoragePath(context.Background(), gameID, globalRoot)
	if err != nil {
		return storage.Mod{}, err
	}

	if err := os.MkdirAll(gameStoragePath, 0o755); err != nil {
		return storage.Mod{}, fmt.Errorf("create game mod storage folder: %w", err)
	}
	if pathContains(gameStoragePath, originalSourcePath) {
		return storage.Mod{}, fmt.Errorf("source folder %q contains the managed mod storage folder %q", originalSourcePath, gameStoragePath)
	}

	destinationPath, err := uniqueManagedModDestination(gameStoragePath, name)
	if err != nil {
		return storage.Mod{}, err
	}

	tempPath, err := makeImportTempDir(gameStoragePath, filepath.Base(destinationPath))
	if err != nil {
		return storage.Mod{}, err
	}
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.RemoveAll(tempPath)
		}
	}()

	if err := copyImportFolder(originalSourcePath, tempPath); err != nil {
		return storage.Mod{}, err
	}

	if _, err := os.Stat(destinationPath); err == nil {
		return storage.Mod{}, fmt.Errorf("managed mod destination %q already exists", destinationPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return storage.Mod{}, fmt.Errorf("check managed mod destination: %w", err)
	}

	if err := os.Rename(tempPath, destinationPath); err != nil {
		return storage.Mod{}, fmt.Errorf("move managed mod folder into place: %w", err)
	}
	removeTemp = false

	removeDestination := true
	defer func() {
		if removeDestination {
			_ = os.RemoveAll(destinationPath)
		}
	}()

	mod, err = s.store.CreateMod(context.Background(), gameID, name, destinationPath, originalSourcePath)
	if err != nil {
		return storage.Mod{}, err
	}

	removeDestination = false
	return mod, nil
}

func normalizeImportedModName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("mod name is required")
	}

	return name, nil
}

func validateImportSourceFolder(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("read source folder: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("source path %q is not a folder", path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read source folder entries: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("source folder %q is empty", path)
	}

	return nil
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

func copyImportFolder(sourcePath string, destinationPath string) error {
	entries, err := os.ReadDir(sourcePath)
	if err != nil {
		return fmt.Errorf("read source folder entries: %w", err)
	}

	for _, entry := range entries {
		sourceEntryPath := filepath.Join(sourcePath, entry.Name())
		destinationEntryPath := filepath.Join(destinationPath, entry.Name())
		if err := copyImportPath(sourceEntryPath, destinationEntryPath); err != nil {
			return err
		}
	}

	return nil
}

func copyImportPath(sourcePath string, destinationPath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("read source path %q: %w", sourcePath, err)
	}

	if info.IsDir() {
		if err := os.Mkdir(destinationPath, info.Mode().Perm()); err != nil {
			return fmt.Errorf("create destination folder %q: %w", destinationPath, err)
		}

		entries, err := os.ReadDir(sourcePath)
		if err != nil {
			return fmt.Errorf("read source folder entries %q: %w", sourcePath, err)
		}

		for _, entry := range entries {
			if err := copyImportPath(filepath.Join(sourcePath, entry.Name()), filepath.Join(destinationPath, entry.Name())); err != nil {
				return err
			}
		}

		return nil
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("source path %q is not a regular file or folder", sourcePath)
	}

	return copyImportFile(sourcePath, destinationPath, info.Mode().Perm())
}

func copyImportFile(sourcePath string, destinationPath string, permissions os.FileMode) (err error) {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open source file %q: %w", sourcePath, err)
	}
	defer func() {
		if closeErr := source.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close source file %q: %w", sourcePath, closeErr)
		}
	}()

	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)
	if err != nil {
		return fmt.Errorf("create destination file %q: %w", destinationPath, err)
	}
	defer func() {
		if closeErr := destination.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close destination file %q: %w", destinationPath, closeErr)
		}
	}()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("copy source file %q: %w", sourcePath, err)
	}

	return nil
}
