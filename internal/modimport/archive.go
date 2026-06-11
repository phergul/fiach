package modimport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/mholt/archives"
	"github.com/nwaples/rardecode/v2"

	"github.com/phergul/fiach/internal/fileignore"
	"github.com/phergul/fiach/internal/storage"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

var windowsVolumePath = regexp.MustCompile(`^[A-Za-z]:`)
var multipartRARName = regexp.MustCompile(`(?i)\.part\d+\.rar$`)
var multipartSevenZipName = regexp.MustCompile(`(?i)\.7z\.\d{3}$`)
var multipartOldRARName = regexp.MustCompile(`(?i)\.r\d{2}$`)

type archiveFormatSpec struct {
	suffix             string
	canonicalExtension string
	label              string
	extractor          archives.Extraction
}

var supportedArchiveFormats = []archiveFormatSpec{
	{suffix: ".tar.bz2", canonicalExtension: ".tar.bz2", label: "tar.bz2", extractor: compressedTar(archives.Bz2{})},
	{suffix: ".tar.gz", canonicalExtension: ".tar.gz", label: "tar.gz", extractor: compressedTar(archives.Gz{})},
	{suffix: ".tar.xz", canonicalExtension: ".tar.xz", label: "tar.xz", extractor: compressedTar(archives.Xz{})},
	{suffix: ".tar.zst", canonicalExtension: ".tar.zst", label: "tar.zst", extractor: compressedTar(archives.Zstd{})},
	{suffix: ".tbz2", canonicalExtension: ".tar.bz2", label: "tar.bz2", extractor: compressedTar(archives.Bz2{})},
	{suffix: ".tgz", canonicalExtension: ".tar.gz", label: "tar.gz", extractor: compressedTar(archives.Gz{})},
	{suffix: ".txz", canonicalExtension: ".tar.xz", label: "tar.xz", extractor: compressedTar(archives.Xz{})},
	{suffix: ".tzst", canonicalExtension: ".tar.zst", label: "tar.zst", extractor: compressedTar(archives.Zstd{})},
	{suffix: ".7z", canonicalExtension: ".7z", label: "7z", extractor: archives.SevenZip{}},
	{suffix: ".rar", canonicalExtension: ".rar", label: "RAR", extractor: archives.Rar{}},
	{suffix: ".tar", canonicalExtension: ".tar", label: "tar", extractor: archives.Tar{}},
	{suffix: ".zip", canonicalExtension: ".zip", label: "zip", extractor: archives.Zip{}},
}

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
	name := s.originalName
	if spec, found := archiveFormatForPath(name); found {
		name = name[:len(name)-len(spec.suffix)]
	} else {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	if strings.TrimSpace(name) == "" {
		return "Imported Mod"
	}

	return name
}

func (s ArchiveSource) Validate(ctx context.Context) error {
	_, err := s.layout(ctx)
	return err
}

func (s ArchiveSource) Materialize(ctx context.Context, destinationPath string) error {
	layout, err := s.layout(ctx)
	if err != nil {
		return err
	}

	return s.walk(ctx, func(_ context.Context, info archives.FileInfo) error {
		entry, ignored, err := archiveEntryFromInfo(info)
		if err != nil {
			return err
		}
		if ignored {
			return nil
		}

		targetName := entry.cleanName
		if layout.stripRoot != "" {
			targetName = strings.TrimPrefix(targetName, layout.stripRoot+"/")
			if targetName == layout.stripRoot {
				return nil
			}
		}
		if targetName == "" || targetName == "." {
			return nil
		}

		destinationEntryPath, err := safeDestinationPath(destinationPath, targetName)
		if err != nil {
			return err
		}

		if entry.isDir {
			if err := os.MkdirAll(destinationEntryPath, archiveDirectoryPermissions(entry.mode)); err != nil {
				return fmt.Errorf("create archive folder %q: %w", destinationEntryPath, err)
			}
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(destinationEntryPath), 0o755); err != nil {
			return fmt.Errorf("create archive file parent %q: %w", filepath.Dir(destinationEntryPath), err)
		}
		if err := extractArchiveFile(ctx, info, destinationEntryPath, entry.mode.Perm()); err != nil {
			return err
		}

		return nil
	})
}

type archiveEntry struct {
	cleanName string
	mode      os.FileMode
	isDir     bool
}

type archiveImportLayout struct {
	entries   []archiveEntry
	stripRoot string
}

func (s ArchiveSource) layout(ctx context.Context) (archiveImportLayout, error) {
	entries := make([]archiveEntry, 0)
	err := s.walk(ctx, func(_ context.Context, info archives.FileInfo) error {
		entry, ignored, err := archiveEntryFromInfo(info)
		if err != nil {
			return err
		}
		if !ignored {
			entries = append(entries, entry)
		}
		return nil
	})
	if err != nil {
		return archiveImportLayout{}, err
	}

	spec, _ := archiveFormatForPath(s.originalPath)
	return archiveLayout(entries, spec.label)
}

func (s ArchiveSource) walk(ctx context.Context, handleFile archives.FileHandler) (err error) {
	spec, err := validateArchivePath(s.originalPath)
	if err != nil {
		return err
	}

	file, err := os.Open(s.originalPath)
	if err != nil {
		return fmt.Errorf("open archive file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close archive file: %w", closeErr)
		}
	}()

	extractor, err := identifyArchiveFormat(ctx, file, spec)
	if err != nil {
		return fmt.Errorf("open %s archive: %w", spec.label, normalizeArchiveError(err))
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("rewind archive file: %w", err)
	}

	if err := extractor.Extract(ctx, file, handleFile); err != nil {
		return fmt.Errorf("open %s archive: %w", spec.label, normalizeArchiveError(err))
	}

	return nil
}

func validateArchivePath(archivePath string) (archiveFormatSpec, error) {
	name := filepath.Base(archivePath)
	if multipartRARName.MatchString(name) || multipartSevenZipName.MatchString(name) || multipartOldRARName.MatchString(name) {
		return archiveFormatSpec{}, errors.New("multipart archives are not supported")
	}

	spec, found := archiveFormatForPath(name)
	if !found {
		return archiveFormatSpec{}, fmt.Errorf("archive path %q has an unsupported archive type", archivePath)
	}

	return spec, nil
}

func archiveFormatForPath(archivePath string) (archiveFormatSpec, bool) {
	name := strings.ToLower(filepath.Base(archivePath))
	for _, spec := range supportedArchiveFormats {
		if strings.HasSuffix(name, spec.suffix) {
			return spec, true
		}
	}

	return archiveFormatSpec{}, false
}

func identifyArchiveFormat(ctx context.Context, file *os.File, expected archiveFormatSpec) (archives.Extraction, error) {
	format, _, err := archives.Identify(ctx, "", file)
	if err != nil {
		if errors.Is(err, archives.NoMatch) {
			if expected.canonicalExtension == ".zip" && fileHasHeader(file, []byte("PK\x05\x06")) {
				return expected.extractor, nil
			}
			if expected.canonicalExtension == ".tar" {
				return expected.extractor, nil
			}
			return nil, errors.New("archive format was not recognized")
		}
		return nil, fmt.Errorf("identify archive format: %w", err)
	}

	extractor, ok := format.(archives.Extraction)
	if !ok {
		if strings.HasPrefix(expected.canonicalExtension, ".tar.") &&
			strings.EqualFold(format.Extension(), strings.TrimPrefix(expected.canonicalExtension, ".tar")) {
			return expected.extractor, nil
		}
		return nil, fmt.Errorf("archive content is %s compression without an archive", format.Extension())
	}
	if !strings.EqualFold(format.Extension(), expected.canonicalExtension) {
		return nil, fmt.Errorf(
			"archive content format %q does not match file extension %q",
			format.Extension(),
			expected.suffix,
		)
	}

	return extractor, nil
}

func fileHasHeader(file *os.File, header []byte) bool {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return false
	}

	buffer := make([]byte, len(header))
	if _, err := io.ReadFull(file, buffer); err != nil {
		return false
	}

	return slices.Equal(buffer, header)
}

func compressedTar(compression archives.Compression) archives.CompressedArchive {
	return archives.CompressedArchive{
		Extraction:  archives.Tar{},
		Compression: compression,
	}
}

func archiveEntryFromInfo(info archives.FileInfo) (archiveEntry, bool, error) {
	cleanName, err := cleanArchiveEntryName(info.NameInArchive)
	if err != nil {
		return archiveEntry{}, false, err
	}
	if archiveEntryIgnored(cleanName) {
		return archiveEntry{}, true, nil
	}

	mode := info.Mode()
	if mode&os.ModeSymlink != 0 {
		return archiveEntry{}, false, fmt.Errorf("archive entry %q is a symlink", info.NameInArchive)
	}
	if info.LinkTarget != "" {
		return archiveEntry{}, false, fmt.Errorf("archive entry %q is a link", info.NameInArchive)
	}

	isDir := info.IsDir()
	if !isDir && !mode.IsRegular() {
		return archiveEntry{}, false, fmt.Errorf("archive entry %q is not a regular file or folder", info.NameInArchive)
	}

	return archiveEntry{
		cleanName: cleanName,
		mode:      mode,
		isDir:     isDir,
	}, false, nil
}

func archiveLayout(entries []archiveEntry, formatLabel string) (archiveImportLayout, error) {
	if len(entries) == 0 {
		return archiveImportLayout{}, fmt.Errorf("%s archive is empty", formatLabel)
	}

	rootNames := map[string]struct{}{}
	hasRootFile := false
	regularFiles := 0

	for _, entry := range entries {
		if !entry.isDir {
			regularFiles++
		}

		root, hasChild := topLevelArchiveName(entry.cleanName)
		if root != "" {
			rootNames[root] = struct{}{}
		}
		if !entry.isDir && !hasChild {
			hasRootFile = true
		}
	}

	if regularFiles == 0 {
		return archiveImportLayout{}, fmt.Errorf("%s archive contains no files", formatLabel)
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

func extractArchiveFile(ctx context.Context, info archives.FileInfo, destinationPath string, permissions os.FileMode) (err error) {
	source, err := info.Open()
	if err != nil {
		return fmt.Errorf("open archive entry %q: %w", info.NameInArchive, err)
	}
	defer func() {
		if closeErr := source.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close archive entry %q: %w", info.NameInArchive, closeErr)
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

	if _, err := io.Copy(destination, contextReader{ctx: ctx, reader: source}); err != nil {
		return fmt.Errorf("extract archive entry %q: %w", info.NameInArchive, err)
	}

	return nil
}

func normalizeArchiveError(err error) error {
	if errors.Is(err, rardecode.ErrArchiveEncrypted) ||
		errors.Is(err, rardecode.ErrArchivedFileEncrypted) ||
		errors.Is(err, rardecode.ErrBadPassword) {
		return errors.New("password-protected archives are not supported")
	}
	if errors.Is(err, rardecode.ErrMultiVolume) {
		return errors.New("multipart archives are not supported")
	}

	var readErr *sevenzip.ReadError
	if errors.As(err, &readErr) && readErr.Encrypted {
		return errors.New("password-protected archives are not supported")
	}

	return err
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(bytes []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(bytes)
}
