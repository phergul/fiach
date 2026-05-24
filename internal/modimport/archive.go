package modimport

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/phergul/mod-manager/internal/fileignore"
	"github.com/phergul/mod-manager/internal/storage"
	"github.com/phergul/mod-manager/internal/storage/dbtypes"
)

var windowsVolumePath = regexp.MustCompile(`^[A-Za-z]:`)

type ArchiveSource struct {
	originalPath string
	originalName string
}

func NewArchiveSource(archiveFilePath string) (source ArchiveSource, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("prepare archive import source: %w", err)
		}
	}()

	originalPath, err := storage.CanonicalModOriginalSourcePath(archiveFilePath)
	if err != nil {
		return ArchiveSource{}, err
	}

	return ArchiveSource{
		originalPath: originalPath,
		originalName: filepath.Base(originalPath),
	}, nil
}

func (s ArchiveSource) Type() dbtypes.ModSourceType {
	return dbtypes.ModSourceTypeArchive
}

func (s ArchiveSource) OriginalPath() string {
	return s.originalPath
}

func (s ArchiveSource) OriginalName() *string {
	if s.originalName == "" {
		return nil
	}

	name := s.originalName
	return &name
}

func (s ArchiveSource) SuggestedName() string {
	name := strings.TrimSuffix(s.originalName, filepath.Ext(s.originalName))
	if strings.TrimSpace(name) == "" {
		return "Imported Mod"
	}

	return name
}

func (s ArchiveSource) Validate() error {
	if !strings.EqualFold(filepath.Ext(s.originalPath), ".zip") {
		return fmt.Errorf("archive path %q is not a .zip file", s.originalPath)
	}

	reader, err := zip.OpenReader(s.originalPath)
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	if _, err := archiveLayout(reader.File); err != nil {
		return err
	}

	return nil
}

func (s ArchiveSource) Materialize(destinationPath string) error {
	reader, err := zip.OpenReader(s.originalPath)
	if err != nil {
		return fmt.Errorf("open zip archive: %w", err)
	}
	defer func() {
		_ = reader.Close()
	}()

	layout, err := archiveLayout(reader.File)
	if err != nil {
		return err
	}

	for _, entry := range layout.entries {
		targetName := entry.cleanName
		if layout.stripRoot != "" {
			targetName = strings.TrimPrefix(targetName, layout.stripRoot+"/")
			if targetName == layout.stripRoot {
				continue
			}
		}
		if targetName == "" || targetName == "." {
			continue
		}

		destinationEntryPath, err := safeDestinationPath(destinationPath, targetName)
		if err != nil {
			return err
		}

		if entry.isDir {
			if err := os.MkdirAll(destinationEntryPath, archiveDirectoryPermissions(entry.mode)); err != nil {
				return fmt.Errorf("create archive folder %q: %w", destinationEntryPath, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destinationEntryPath), 0o755); err != nil {
			return fmt.Errorf("create archive file parent %q: %w", filepath.Dir(destinationEntryPath), err)
		}
		if err := extractArchiveFile(entry.file, destinationEntryPath, entry.mode.Perm()); err != nil {
			return err
		}
	}

	return nil
}

type archiveEntry struct {
	file      *zip.File
	cleanName string
	mode      os.FileMode
	isDir     bool
}

type archiveImportLayout struct {
	entries   []archiveEntry
	stripRoot string
}

func archiveLayout(files []*zip.File) (archiveImportLayout, error) {
	if len(files) == 0 {
		return archiveImportLayout{}, errors.New("zip archive is empty")
	}

	entries := make([]archiveEntry, 0, len(files))
	rootNames := map[string]struct{}{}
	hasRootFile := false
	regularFiles := 0

	for _, file := range files {
		cleanName, err := cleanArchiveEntryName(file.Name)
		if err != nil {
			return archiveImportLayout{}, err
		}
		if archiveEntryIgnored(cleanName) {
			continue
		}

		mode := file.FileInfo().Mode()
		isDir := file.FileInfo().IsDir()
		if mode&os.ModeSymlink != 0 {
			return archiveImportLayout{}, fmt.Errorf("archive entry %q is a symlink", file.Name)
		}
		if !isDir && !mode.IsRegular() {
			return archiveImportLayout{}, fmt.Errorf("archive entry %q is not a regular file or folder", file.Name)
		}
		if !isDir {
			regularFiles++
		}

		root, hasChild := topLevelArchiveName(cleanName)
		if root != "" {
			rootNames[root] = struct{}{}
		}
		if !isDir && !hasChild {
			hasRootFile = true
		}

		entries = append(entries, archiveEntry{
			file:      file,
			cleanName: cleanName,
			mode:      mode,
			isDir:     isDir,
		})
	}

	if regularFiles == 0 {
		return archiveImportLayout{}, errors.New("zip archive contains no files")
	}

	stripRoot := ""
	if len(rootNames) == 1 && !hasRootFile {
		for rootName := range rootNames {
			stripRoot = rootName
		}
	}

	return archiveImportLayout{
		entries:   entries,
		stripRoot: stripRoot,
	}, nil
}

func cleanArchiveEntryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("archive entry path is empty")
	}
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(name, "/") || path.IsAbs(name) || filepath.IsAbs(name) || windowsVolumePath.MatchString(name) {
		return "", fmt.Errorf("archive entry %q is an absolute path", name)
	}

	cleanName := path.Clean(name)
	if cleanName == "." || cleanName == "/" || cleanName == "" {
		return "", fmt.Errorf("archive entry %q is not a valid path", name)
	}
	if cleanName == ".." || strings.HasPrefix(cleanName, "../") || strings.Contains(cleanName, "/../") {
		return "", fmt.Errorf("archive entry %q escapes the archive root", name)
	}

	return cleanName, nil
}

func topLevelArchiveName(name string) (root string, hasChild bool) {
	parts := strings.SplitN(name, "/", 2)
	if len(parts) == 0 {
		return "", false
	}
	if len(parts) == 1 {
		return parts[0], false
	}

	return parts[0], true
}

func archiveEntryIgnored(name string) bool {
	return slices.ContainsFunc(strings.Split(name, "/"), fileignore.Has)
}

func safeDestinationPath(root string, entryName string) (string, error) {
	destinationPath := filepath.Join(root, filepath.FromSlash(entryName))
	relativePath, err := filepath.Rel(root, destinationPath)
	if err != nil {
		return "", fmt.Errorf("resolve archive destination %q: %w", entryName, err)
	}
	if relativePath == "." || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry %q escapes the managed mod folder", entryName)
	}

	return destinationPath, nil
}

func archiveDirectoryPermissions(mode os.FileMode) os.FileMode {
	permissions := mode.Perm()
	if permissions == 0 {
		return 0o755
	}

	return permissions
}

func extractArchiveFile(file *zip.File, destinationPath string, permissions os.FileMode) (err error) {
	source, err := file.Open()
	if err != nil {
		return fmt.Errorf("open archive entry %q: %w", file.Name, err)
	}
	defer func() {
		if closeErr := source.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close archive entry %q: %w", file.Name, closeErr)
		}
	}()

	if permissions == 0 {
		permissions = 0o644
	}
	destination, err := os.OpenFile(destinationPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, permissions)
	if err != nil {
		return fmt.Errorf("create archive file %q: %w", destinationPath, err)
	}
	defer func() {
		if closeErr := destination.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close archive file %q: %w", destinationPath, closeErr)
		}
	}()

	if _, err := io.Copy(destination, source); err != nil {
		return fmt.Errorf("extract archive entry %q: %w", file.Name, err)
	}

	return nil
}
