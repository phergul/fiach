package optiscaler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/mholt/archives"

	"github.com/phergul/fiach/internal/fileops"
)

var windowsArchiveVolume = regexp.MustCompile(`^[A-Za-z]:`)

type PackageFile struct {
	RelativePath string `json:"relativePath"`
	SourcePath   string `json:"sourcePath"`
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"sizeBytes"`
}

type Package struct {
	Root  string        `json:"root"`
	Files []PackageFile `json:"files"`
}

func ExtractReleasePackage(ctx context.Context, archivePath string, destination string) (pkg Package, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("extract verified OptiScaler package: %w", err)
		}
	}()

	if err := os.RemoveAll(destination); err != nil {
		return Package{}, err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return Package{}, err
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return Package{}, err
	}
	defer file.Close()
	format, _, err := archives.Identify(ctx, filepath.Base(archivePath), file)
	if err != nil {
		return Package{}, err
	}
	extractor, ok := format.(archives.Extraction)
	if !ok {
		return Package{}, errors.New("release archive format is not extractable")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return Package{}, err
	}
	seen := map[string]bool{}
	err = extractor.Extract(ctx, file, func(ctx context.Context, info archives.FileInfo) error {
		cleanName, err := cleanPackageEntry(info.NameInArchive)
		if err != nil {
			return err
		}
		lowerName := strings.ToLower(cleanName)
		if seen[lowerName] {
			return fmt.Errorf("archive contains a case-insensitive duplicate path %q", cleanName)
		}
		seen[lowerName] = true
		if info.LinkTarget != "" || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %q is a link", cleanName)
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return fmt.Errorf("archive entry %q is a special file", cleanName)
		}
		target := filepath.Join(destination, filepath.FromSlash(cleanName))
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		source, err := info.Open()
		if err != nil {
			return err
		}
		defer source.Close()
		output, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(output, source)
		closeErr := output.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		return Package{}, err
	}

	files, err := packageInventory(destination)
	if err != nil {
		return Package{}, err
	}
	return Package{
		Root:  destination,
		Files: files,
	}, nil
}

func cleanPackageEntry(name string) (string, error) {
	name = strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	if name == "" || strings.HasPrefix(name, "/") || path.IsAbs(name) || windowsArchiveVolume.MatchString(name) {
		return "", fmt.Errorf("archive entry %q is absolute or empty", name)
	}
	cleanName := path.Clean(name)
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, "../") {
		return "", fmt.Errorf("archive entry %q escapes the archive root", name)
	}
	return cleanName, nil
}

func packageInventory(root string) ([]PackageFile, error) {
	var files []PackageFile
	hasRuntime := false
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if !includePackageFile(relative) {
			return nil
		}
		hash, size, err := fileops.FileIntegrity(path)
		if err != nil {
			return err
		}
		hasRuntime = hasRuntime || strings.EqualFold(filepath.Base(relative), "OptiScaler.dll")
		files = append(files, PackageFile{
			RelativePath: filepath.Clean(relative),
			SourcePath:   path,
			SHA256:       hash,
			SizeBytes:    size,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !hasRuntime {
		return nil, errors.New("package does not contain OptiScaler.dll")
	}
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].RelativePath) < strings.ToLower(files[j].RelativePath)
	})
	return files, nil
}

func includePackageFile(relative string) bool {
	name := strings.ToLower(filepath.Base(filepath.Clean(relative)))
	if strings.HasSuffix(name, ".bat") ||
		strings.HasSuffix(name, ".cmd") ||
		strings.HasSuffix(name, ".sh") {
		return false
	}
	if strings.Contains(name, "readme") &&
		(strings.HasSuffix(name, ".txt") || strings.HasSuffix(name, ".md")) {
		return false
	}
	if strings.HasSuffix(name, ".txt") &&
		(strings.Contains(name, "setup") || strings.Contains(name, "install")) {
		return false
	}
	return true
}
