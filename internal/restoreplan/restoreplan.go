package restoreplan

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/phergul/mod-manager/internal/appliedstate"
)

type RestoreOperationType string

const (
	RestoreOperationTypeRemoveAddedFile      RestoreOperationType = "remove_added_file"
	RestoreOperationTypeRestoreReplacedFile  RestoreOperationType = "restore_replaced_file"
	RestoreOperationTypeRemoveCreatedDir     RestoreOperationType = "remove_created_directory"
	RestoreOperationTypeDeleteRestoredBackup RestoreOperationType = "delete_restored_backup"
)

type RestoreOperationStatus string

const (
	RestoreOperationStatusCompleted RestoreOperationStatus = "completed"
	RestoreOperationStatusFailed    RestoreOperationStatus = "failed"
	RestoreOperationStatusSkipped   RestoreOperationStatus = "skipped"
)

type RestoreOperation struct {
	Type                   RestoreOperationType
	ManifestOperationIndex int
	Mod                    Mod
	TargetPath             string
	BackupPath             *string
}

type Mod struct {
	ID   int64
	Name string
}

type RestoreOperationResult struct {
	OperationIndex int
	Operation      RestoreOperation
	Status         RestoreOperationStatus
	Message        string
	Error          *string
}

type RestoreResult struct {
	Success        bool
	CompletedCount int
	FailedCount    int
	SkippedCount   int
	Results        []RestoreOperationResult
}

type Context struct {
	GameInstallPath    string
	GameModStoragePath string
}

type resolvedContext struct {
	gameInstallPath    string
	gameModStoragePath string
}

var computeFileIntegrity = fileIntegrity

func Execute(manifest appliedstate.ManifestDocument, context Context) (result RestoreResult, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("execute restore plan: %w", err)
		}
	}()

	resolved, err := validateContext(context)
	if err != nil {
		return RestoreResult{}, err
	}

	operations := buildOperations(manifest)
	if failures := preflightOperations(operations, manifest, resolved); len(failures) > 0 {
		return failedPreflightResult(operations, failures), nil
	}

	result.Success = true
	result.Results = make([]RestoreOperationResult, 0, len(operations))
	for index, operation := range operations {
		message, operationErr := executeOperation(operation, resolved)
		if operationErr != nil {
			result.Success = false
			result.Results = append(result.Results, newFailedResult(index, operation, operationErr))
			appendSkippedResults(operations, index+1, &result)
			updateCounts(&result)
			return result, nil
		}

		result.Results = append(result.Results, RestoreOperationResult{
			OperationIndex: index,
			Operation:      operation,
			Status:         RestoreOperationStatusCompleted,
			Message:        message,
		})
	}

	updateCounts(&result)
	return result, nil
}

func buildOperations(manifest appliedstate.ManifestDocument) []RestoreOperation {
	operations := make([]RestoreOperation, 0, len(manifest.AddedFiles)+len(manifest.ReplacedFiles)*2+len(manifest.CreatedDirectories))

	for _, entry := range manifest.AddedFiles {
		operations = append(operations, RestoreOperation{
			Type:                   RestoreOperationTypeRemoveAddedFile,
			ManifestOperationIndex: entry.OperationIndex,
			Mod:                    modFromManifest(entry.Mod),
			TargetPath:             entry.TargetPath,
		})
	}
	for _, entry := range manifest.ReplacedFiles {
		backupPath := entry.BackupPath
		operations = append(operations, RestoreOperation{
			Type:                   RestoreOperationTypeRestoreReplacedFile,
			ManifestOperationIndex: entry.OperationIndex,
			Mod:                    modFromManifest(entry.Mod),
			TargetPath:             entry.TargetPath,
			BackupPath:             &backupPath,
		})
	}

	directories := append([]appliedstate.CreatedDirectory(nil), manifest.CreatedDirectories...)
	sort.SliceStable(directories, func(i int, j int) bool {
		iPath := cleanPathForSort(directories[i].TargetPath)
		jPath := cleanPathForSort(directories[j].TargetPath)
		iDepth := pathDepth(iPath)
		jDepth := pathDepth(jPath)
		if iDepth != jDepth {
			return iDepth > jDepth
		}

		return iPath > jPath
	})
	for _, entry := range directories {
		operations = append(operations, RestoreOperation{
			Type:                   RestoreOperationTypeRemoveCreatedDir,
			ManifestOperationIndex: entry.OperationIndex,
			Mod:                    modFromManifest(entry.Mod),
			TargetPath:             entry.TargetPath,
		})
	}

	for _, entry := range manifest.ReplacedFiles {
		backupPath := entry.BackupPath
		operations = append(operations, RestoreOperation{
			Type:                   RestoreOperationTypeDeleteRestoredBackup,
			ManifestOperationIndex: entry.OperationIndex,
			Mod:                    modFromManifest(entry.Mod),
			TargetPath:             entry.TargetPath,
			BackupPath:             &backupPath,
		})
	}

	return operations
}

func modFromManifest(mod appliedstate.Mod) Mod {
	return Mod{
		ID:   mod.ID,
		Name: mod.Name,
	}
}

func validateContext(context Context) (resolvedContext, error) {
	gameInstallPath, err := cleanRootPath("game install path", context.GameInstallPath)
	if err != nil {
		return resolvedContext{}, err
	}
	gameModStoragePath, err := cleanRootPath("game mod storage path", context.GameModStoragePath)
	if err != nil {
		return resolvedContext{}, err
	}

	return resolvedContext{
		gameInstallPath:    gameInstallPath,
		gameModStoragePath: gameModStoragePath,
	}, nil
}

func preflightOperations(operations []RestoreOperation, manifest appliedstate.ManifestDocument, context resolvedContext) map[int]error {
	failures := map[int]error{}
	addedFiles := map[int]appliedstate.AddedFile{}
	replacedFiles := map[int]appliedstate.ReplacedFile{}
	createdDirectories := map[int]appliedstate.CreatedDirectory{}

	for _, entry := range manifest.AddedFiles {
		addedFiles[entry.OperationIndex] = entry
	}
	for _, entry := range manifest.ReplacedFiles {
		replacedFiles[entry.OperationIndex] = entry
	}
	for _, entry := range manifest.CreatedDirectories {
		createdDirectories[entry.OperationIndex] = entry
	}

	for index, operation := range operations {
		var err error
		switch operation.Type {
		case RestoreOperationTypeRemoveAddedFile:
			err = preflightAddedFile(addedFiles[operation.ManifestOperationIndex], context)
		case RestoreOperationTypeRestoreReplacedFile:
			err = preflightReplacedFile(replacedFiles[operation.ManifestOperationIndex], context)
		case RestoreOperationTypeRemoveCreatedDir:
			err = preflightCreatedDirectory(createdDirectories[operation.ManifestOperationIndex], context)
		case RestoreOperationTypeDeleteRestoredBackup:
			err = preflightBackupCleanup(replacedFiles[operation.ManifestOperationIndex], context)
		default:
			err = fmt.Errorf("unsupported restore operation type %q", operation.Type)
		}
		if err != nil {
			failures[index] = err
		}
	}

	return failures
}

func preflightAddedFile(entry appliedstate.AddedFile, context resolvedContext) error {
	targetPath, err := cleanManifestPath("added file target path", entry.TargetPath)
	if err != nil {
		return err
	}
	if err := requirePathWithinRoot("added file target path", targetPath, context.gameInstallPath); err != nil {
		return err
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat added file target %q: %w", targetPath, err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("added file target %q is not a regular file", targetPath)
	}
	if err := requireFileIntegrity(targetPath, entry.SHA256, entry.SizeBytes, "added file target"); err != nil {
		return err
	}

	return nil
}

func preflightReplacedFile(entry appliedstate.ReplacedFile, context resolvedContext) error {
	targetPath, err := cleanManifestPath("replaced file target path", entry.TargetPath)
	if err != nil {
		return err
	}
	if err := requirePathWithinRoot("replaced file target path", targetPath, context.gameInstallPath); err != nil {
		return err
	}
	backupPath, err := cleanManifestPath("backup file path", entry.BackupPath)
	if err != nil {
		return err
	}
	if err := requirePathWithinRoot("backup file path", backupPath, context.gameModStoragePath); err != nil {
		return err
	}

	info, err := statRegularFile("replaced file target", targetPath)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("replaced file target %q is not a regular file", targetPath)
	}

	moddedMatch, moddedErr := fileMatchesIntegrity(targetPath, entry.SHA256, entry.SizeBytes)
	backupMatch, backupErr := fileMatchesIntegrity(targetPath, entry.BackupSHA256, entry.BackupSizeBytes)
	if moddedErr != nil {
		return fmt.Errorf("read replaced file target integrity %q: %w", targetPath, moddedErr)
	}
	if backupErr != nil {
		return fmt.Errorf("read restored file target integrity %q: %w", targetPath, backupErr)
	}
	if !moddedMatch && !backupMatch {
		return fmt.Errorf("replaced file target %q does not match the applied file or recorded backup integrity", targetPath)
	}
	if err := requireFileIntegrity(backupPath, entry.BackupSHA256, entry.BackupSizeBytes, "backup file"); err != nil {
		return err
	}

	return nil
}

func preflightCreatedDirectory(entry appliedstate.CreatedDirectory, context resolvedContext) error {
	targetPath, err := cleanManifestPath("created directory target path", entry.TargetPath)
	if err != nil {
		return err
	}
	if err := requirePathWithinRoot("created directory target path", targetPath, context.gameInstallPath); err != nil {
		return err
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("stat created directory %q: %w", targetPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("created directory target %q is not a directory", targetPath)
	}

	return nil
}

func preflightBackupCleanup(entry appliedstate.ReplacedFile, context resolvedContext) error {
	backupPath, err := cleanManifestPath("backup file path", entry.BackupPath)
	if err != nil {
		return err
	}
	if err := requirePathWithinRoot("backup file path", backupPath, context.gameModStoragePath); err != nil {
		return err
	}

	return nil
}

func executeOperation(operation RestoreOperation, context resolvedContext) (string, error) {
	switch operation.Type {
	case RestoreOperationTypeRemoveAddedFile:
		return removeAddedFile(operation.TargetPath)
	case RestoreOperationTypeRestoreReplacedFile:
		return restoreReplacedFile(operation.TargetPath, *operation.BackupPath)
	case RestoreOperationTypeRemoveCreatedDir:
		return removeCreatedDirectory(operation.TargetPath)
	case RestoreOperationTypeDeleteRestoredBackup:
		return deleteRestoredBackup(*operation.BackupPath, context.gameModStoragePath)
	default:
		return "", fmt.Errorf("unsupported restore operation type %q", operation.Type)
	}
}

func removeAddedFile(targetPath string) (string, error) {
	if err := os.Remove(targetPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "Added file already absent.", nil
		}
		return "", fmt.Errorf("remove added file %q: %w", targetPath, err)
	}

	return "Removed added file.", nil
}

func restoreReplacedFile(targetPath string, backupPath string) (string, error) {
	backupInfo, err := statRegularFile("backup file", backupPath)
	if err != nil {
		return "", err
	}
	if err := copyFileAtomic(backupPath, targetPath, backupInfo.Mode().Perm()); err != nil {
		return "", err
	}

	return "Restored replaced file from backup.", nil
}

func removeCreatedDirectory(targetPath string) (string, error) {
	err := os.Remove(targetPath)
	if err == nil {
		return "Removed empty created directory.", nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return "Created directory already absent.", nil
	}
	if isDirectoryNotEmptyError(err) {
		return "Created directory was not empty and was left in place.", nil
	}

	return "", fmt.Errorf("remove created directory %q: %w", targetPath, err)
}

func deleteRestoredBackup(backupPath string, gameModStoragePath string) (string, error) {
	if err := os.Remove(backupPath); err != nil {
		return "", fmt.Errorf("delete restored backup %q: %w", backupPath, err)
	}
	if err := removeEmptyParentDirectories(filepath.Dir(backupPath), gameModStoragePath); err != nil {
		return "", err
	}

	return "Deleted restored backup.", nil
}

func failedPreflightResult(operations []RestoreOperation, failures map[int]error) RestoreResult {
	result := RestoreResult{
		Success: false,
		Results: make([]RestoreOperationResult, 0, len(operations)),
	}
	for index, operation := range operations {
		if err, failed := failures[index]; failed {
			result.Results = append(result.Results, newFailedResult(index, operation, err))
			continue
		}

		result.Results = append(result.Results, RestoreOperationResult{
			OperationIndex: index,
			Operation:      operation,
			Status:         RestoreOperationStatusSkipped,
			Message:        "Skipped because restore preflight failed.",
		})
	}
	updateCounts(&result)

	return result
}

func newFailedResult(index int, operation RestoreOperation, err error) RestoreOperationResult {
	errorMessage := err.Error()
	return RestoreOperationResult{
		OperationIndex: index,
		Operation:      operation,
		Status:         RestoreOperationStatusFailed,
		Message:        "Restore operation failed.",
		Error:          &errorMessage,
	}
}

func appendSkippedResults(operations []RestoreOperation, startIndex int, result *RestoreResult) {
	for index := startIndex; index < len(operations); index++ {
		result.Results = append(result.Results, RestoreOperationResult{
			OperationIndex: index,
			Operation:      operations[index],
			Status:         RestoreOperationStatusSkipped,
			Message:        "Skipped after a previous restore operation failed.",
		})
	}
}

func updateCounts(result *RestoreResult) {
	result.CompletedCount = 0
	result.FailedCount = 0
	result.SkippedCount = 0

	for _, operationResult := range result.Results {
		switch operationResult.Status {
		case RestoreOperationStatusCompleted:
			result.CompletedCount++
		case RestoreOperationStatusFailed:
			result.FailedCount++
		case RestoreOperationStatusSkipped:
			result.SkippedCount++
		}
	}
}

func requireFileIntegrity(path string, sha256Hex string, sizeBytes int64, label string) error {
	matches, err := fileMatchesIntegrity(path, sha256Hex, sizeBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s %q is missing", label, path)
		}
		return fmt.Errorf("read %s integrity %q: %w", label, path, err)
	}
	if !matches {
		return fmt.Errorf("%s %q does not match recorded integrity", label, path)
	}

	return nil
}

func fileMatchesIntegrity(path string, sha256Hex string, sizeBytes int64) (bool, error) {
	hash, size, err := computeFileIntegrity(path)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(hash, sha256Hex) && size == sizeBytes, nil
}

func cleanRootPath(name string, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s is required", name)
	}

	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", name, path, err)
	}

	return filepath.Clean(absolutePath), nil
}

func cleanManifestPath(name string, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", name, path, err)
	}

	return filepath.Clean(absolutePath), nil
}

func requirePathWithinRoot(name string, path string, root string) error {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve %s %q: %w", name, path, err)
	}
	cleanPath := filepath.Clean(absolutePath)

	relativePath, err := filepath.Rel(root, cleanPath)
	if err != nil {
		return fmt.Errorf("compare %s %q with root %q: %w", name, cleanPath, root, err)
	}
	if relativePath == "." {
		return nil
	}
	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%s %q is outside %q", name, cleanPath, root)
	}

	return nil
}

func fileIntegrity(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, fmt.Errorf("hash file %q: %w", path, err)
	}

	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func statRegularFile(label string, path string) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s %q is missing", label, path)
		}
		return nil, fmt.Errorf("stat %s %q: %w", label, path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s %q is not a regular file", label, path)
	}

	return info, nil
}

func copyFileAtomic(sourcePath string, targetPath string, mode fs.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open backup file %q: %w", sourcePath, err)
	}
	defer source.Close()

	targetDirectory := filepath.Dir(targetPath)
	tempFile, err := os.CreateTemp(targetDirectory, ".mod-manager-restore-*")
	if err != nil {
		return fmt.Errorf("create temporary file in %q: %w", targetDirectory, err)
	}
	tempPath := tempFile.Name()
	shouldRemoveTemp := true
	defer func() {
		if shouldRemoveTemp {
			_ = os.Remove(tempPath)
		}
	}()

	if _, err := io.Copy(tempFile, source); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("copy %q to temporary file %q: %w", sourcePath, tempPath, err)
	}
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("set temporary file mode %q: %w", tempPath, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary file %q: %w", tempPath, err)
	}

	if err := os.Rename(tempPath, targetPath); err == nil {
		shouldRemoveTemp = false
		return nil
	}
	if err := os.Remove(targetPath); err != nil {
		return fmt.Errorf("remove existing target file %q: %w", targetPath, err)
	}
	if err := os.Rename(tempPath, targetPath); err != nil {
		return fmt.Errorf("move temporary file %q to %q: %w", tempPath, targetPath, err)
	}
	shouldRemoveTemp = false
	return nil
}

func removeEmptyParentDirectories(startPath string, root string) error {
	current := filepath.Clean(startPath)
	root = filepath.Clean(root)

	for {
		if current == root {
			return nil
		}
		if err := requirePathWithinRoot("backup parent directory", current, root); err != nil {
			return err
		}

		err := os.Remove(current)
		if err == nil {
			current = filepath.Dir(current)
			continue
		}
		if errors.Is(err, os.ErrNotExist) || isDirectoryNotEmptyError(err) {
			return nil
		}
		return fmt.Errorf("remove empty backup directory %q: %w", current, err)
	}
}

func cleanPathForSort(path string) string {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}

	return filepath.Clean(absolutePath)
}

func pathDepth(path string) int {
	volume := filepath.VolumeName(path)
	path = strings.TrimPrefix(path, volume)
	path = strings.Trim(path, string(filepath.Separator))
	if path == "" {
		return 0
	}

	return len(strings.Split(path, string(filepath.Separator)))
}

func isDirectoryNotEmptyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, syscall.ENOTEMPTY) || errors.Is(err, syscall.EEXIST) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "directory not empty") ||
		strings.Contains(message, "not empty")
}
