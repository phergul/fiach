package fileops

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"
)

type AtomicCopyOptions struct {
	SourcePath string
	TargetPath string
	Mode       fs.FileMode
	Replace    bool
	TempPrefix string
	OpenLabel  string
}

func CopyFileAtomic(options AtomicCopyOptions) error {
	source, err := os.Open(options.SourcePath)
	if err != nil {
		label := strings.TrimSpace(options.OpenLabel)
		if label == "" {
			label = "source file"
		}
		return fmt.Errorf("open %s %q: %w", label, options.SourcePath, err)
	}
	defer source.Close()

	targetDirectory := filepath.Dir(options.TargetPath)
	tempPrefix := options.TempPrefix
	if tempPrefix == "" {
		tempPrefix = ".fiach-*"
	}
	tempFile, err := os.CreateTemp(targetDirectory, tempPrefix)
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
		return fmt.Errorf("copy %q to temporary file %q: %w", options.SourcePath, tempPath, err)
	}
	if err := tempFile.Chmod(options.Mode); err != nil {
		_ = tempFile.Close()
		return fmt.Errorf("set temporary file mode %q: %w", tempPath, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("close temporary file %q: %w", tempPath, err)
	}

	if options.Replace {
		if err := os.Rename(tempPath, options.TargetPath); err == nil {
			shouldRemoveTemp = false
			return nil
		}
		if err := os.Remove(options.TargetPath); err != nil {
			return fmt.Errorf("remove existing target file %q: %w", options.TargetPath, err)
		}
	} else if _, err := os.Lstat(options.TargetPath); err == nil {
		return fmt.Errorf("target file %q already exists", options.TargetPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat target file %q: %w", options.TargetPath, err)
	}

	if err := os.Rename(tempPath, options.TargetPath); err != nil {
		return fmt.Errorf("move temporary file %q to %q: %w", tempPath, options.TargetPath, err)
	}
	shouldRemoveTemp = false
	return nil
}

func CleanRequiredAbsPath(name string, path string) (string, error) {
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

func CleanAbsPath(name string, path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s %q: %w", name, path, err)
	}

	return filepath.Clean(absolutePath), nil
}

func RequirePathWithinRoot(name string, path string, root string) error {
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

func HashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func HashJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return HashBytes(encoded), nil
}

func HashParts(values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func IsUTF8Text(contents []byte) bool {
	return utf8.Valid(contents) && bytes.IndexByte(contents, 0) < 0
}

func FileIntegrity(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open file %q: %w", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	size, err := io.Copy(hash, file)
	if err != nil {
		return "", 0, fmt.Errorf("hash file %q: %w", path, err)
	}

	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func FileMatchesIntegrity(path string, sha256Hex string, sizeBytes int64) (bool, error) {
	hash, size, err := FileIntegrity(path)
	if err != nil {
		return false, err
	}

	return strings.EqualFold(hash, sha256Hex) && size == sizeBytes, nil
}

func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("stat file %q: %w", path, err)
}

func RenameIfExists(from string, to string) error {
	if err := os.Rename(from, to); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("rename %q to %q: %w", from, to, err)
	}

	return nil
}

func StatRegularFile(label string, path string) (fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s %q does not exist", label, path)
		}
		return nil, fmt.Errorf("stat %s %q: %w", label, path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s %q is not a regular file", label, path)
	}

	return info, nil
}

func ValidateDirEntryIsRegularFile(label string, entry fs.DirEntry) (fs.FileInfo, error) {
	info, err := entry.Info()
	if err != nil {
		return nil, fmt.Errorf("stat %s %q: %w", label, entry.Name(), err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s %q is not a regular file", label, entry.Name())
	}

	return info, nil
}

func RemoveEmptyParentDirectories(startPath string, root string) error {
	current := filepath.Clean(startPath)
	root = filepath.Clean(root)

	for {
		if current == root {
			return nil
		}
		if err := RequirePathWithinRoot("backup parent directory", current, root); err != nil {
			return err
		}

		err := os.Remove(current)
		if err == nil {
			current = filepath.Dir(current)
			continue
		}
		if errors.Is(err, os.ErrNotExist) || IsDirectoryNotEmptyError(err) {
			return nil
		}
		return fmt.Errorf("remove empty backup directory %q: %w", current, err)
	}
}

func IsDirectoryNotEmptyError(err error) bool {
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
