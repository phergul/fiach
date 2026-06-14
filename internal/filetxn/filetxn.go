package filetxn

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
)

type Operation struct {
	Type       string `json:"type"`
	SourcePath string `json:"sourcePath,omitempty"`
	TargetPath string `json:"targetPath"`
	BackupPath string `json:"backupPath,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	SizeBytes  int64  `json:"sizeBytes,omitempty"`
}

type Snapshot struct {
	TargetPath string `json:"targetPath"`
	Existed    bool   `json:"existed"`
	BackupPath string `json:"backupPath,omitempty"`
	SHA256     string `json:"sha256,omitempty"`
	SizeBytes  int64  `json:"sizeBytes,omitempty"`
}

func ValidateOperations(operations []Operation, allowedRoots ...string) error {
	for _, operation := range operations {
		if operation.TargetPath == "" {
			return errors.New("operation target path is required")
		}
		if operation.Type != "copy" && operation.Type != "restore" &&
			operation.Type != "delete" && operation.Type != "move" &&
			operation.Type != "adopt" {
			return fmt.Errorf("unsupported operation type %q", operation.Type)
		}
		if (operation.Type == "copy" || operation.Type == "restore" || operation.Type == "move") &&
			operation.SourcePath == "" {
			return fmt.Errorf("%s operation source path is required", operation.Type)
		}
		if len(allowedRoots) > 0 && !pathWithinAnyRoot(operation.TargetPath, allowedRoots) {
			return fmt.Errorf("operation target path %q is outside allowed roots", operation.TargetPath)
		}
		if operation.Type == "move" && len(allowedRoots) > 0 &&
			!pathWithinAnyRoot(operation.SourcePath, allowedRoots) {
			return fmt.Errorf("move operation source path %q is outside allowed roots", operation.SourcePath)
		}
	}
	return nil
}

func SnapshotOperations(root string, operations []Operation) (snapshots []Snapshot, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("snapshot transaction targets: %w", err)
		}
	}()

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, operation := range operations {
		for _, target := range TouchedPaths(operation) {
			key := strings.ToLower(filepath.Clean(target))
			if seen[key] {
				continue
			}
			seen[key] = true
			snapshot := Snapshot{TargetPath: target}
			info, statErr := os.Stat(target)
			if errors.Is(statErr, os.ErrNotExist) {
				snapshots = append(snapshots, snapshot)
				continue
			}
			if statErr != nil {
				return nil, statErr
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("mutation target %q is not a regular file", target)
			}
			snapshot.Existed = true
			snapshot.BackupPath = filepath.Join(root, fmt.Sprintf("%03d.bak", len(snapshots)))
			if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
				SourcePath: target,
				TargetPath: snapshot.BackupPath,
				Mode:       0o644,
				Replace:    false,
				OpenLabel:  "mutation target",
			}); err != nil {
				return nil, err
			}
			snapshot.SHA256, snapshot.SizeBytes, err = fileops.FileIntegrity(snapshot.BackupPath)
			if err != nil {
				return nil, err
			}
			snapshots = append(snapshots, snapshot)
		}
	}
	return snapshots, nil
}

func TouchedPaths(operation Operation) []string {
	if operation.Type == "move" {
		return []string{operation.SourcePath, operation.TargetPath}
	}
	return []string{operation.TargetPath}
}

func ExecuteOperation(operation Operation, openLabel string) error {
	switch operation.Type {
	case "copy", "restore":
		if err := os.MkdirAll(filepath.Dir(operation.TargetPath), 0o755); err != nil {
			return err
		}
		return fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
			SourcePath: operation.SourcePath,
			TargetPath: operation.TargetPath,
			Mode:       0o644,
			Replace:    true,
			OpenLabel:  openLabel,
		})
	case "delete":
		if err := os.Remove(operation.TargetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	case "move":
		if err := os.Remove(operation.TargetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return os.Rename(operation.SourcePath, operation.TargetPath)
	case "adopt":
		return nil
	default:
		return fmt.Errorf("unsupported operation type %q", operation.Type)
	}
}

func VerifyOperations(operations []Operation) error {
	for _, operation := range operations {
		switch operation.Type {
		case "copy", "restore", "adopt":
			matches, err := fileops.FileMatchesIntegrity(operation.TargetPath, operation.SHA256, operation.SizeBytes)
			if err != nil || !matches {
				return fmt.Errorf("verify managed file %q", operation.TargetPath)
			}
		case "delete":
			if _, err := os.Stat(operation.TargetPath); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("verify deleted file %q", operation.TargetPath)
			}
		case "move":
			info, err := os.Stat(operation.TargetPath)
			if err != nil || !info.Mode().IsRegular() {
				return fmt.Errorf("verify moved target %q", operation.TargetPath)
			}
		}
	}
	return nil
}

func RollbackSnapshots(snapshots []Snapshot) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("rollback transaction snapshots: %w", err)
		}
	}()

	for index := len(snapshots) - 1; index >= 0; index-- {
		snapshot := snapshots[index]
		if snapshot.Existed {
			if err := os.MkdirAll(filepath.Dir(snapshot.TargetPath), 0o755); err != nil {
				return err
			}
			if err := fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
				SourcePath: snapshot.BackupPath,
				TargetPath: snapshot.TargetPath,
				Mode:       0o644,
				Replace:    true,
				OpenLabel:  "journal backup",
			}); err != nil {
				return err
			}
			matches, err := fileops.FileMatchesIntegrity(snapshot.TargetPath, snapshot.SHA256, snapshot.SizeBytes)
			if err != nil || !matches {
				return fmt.Errorf("restored file %q failed verification", snapshot.TargetPath)
			}
		} else if err := os.Remove(snapshot.TargetPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func WriteJSONAtomic(path string, value any) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("write transaction journal: %w", err)
		}
	}()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".journal-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(contents); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tempPath, path)
}

func pathWithinAnyRoot(path string, roots []string) bool {
	for _, root := range roots {
		if fileops.RequirePathWithinRoot("operation target", path, root) == nil {
			return true
		}
	}
	return false
}
