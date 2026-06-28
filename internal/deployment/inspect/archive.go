package inspect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mholt/archives"
	"github.com/phergul/fiach/internal/fileignore"
)

var archiveWindowsVolumePath = regexp.MustCompile(`^[A-Za-z]:`)

type archiveListResult struct {
	Entries      []ArchiveEntry
	LimitReached bool
	LimitReason  string
}

func listArchive(path string) (archiveListResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return archiveListResult{}, fmt.Errorf("open archive file %q: %w", path, err)
	}
	defer file.Close()

	format, _, err := archives.Identify(context.Background(), filepath.Base(path), file)
	if err != nil {
		if errors.Is(err, archives.NoMatch) {
			return archiveListResult{}, fmt.Errorf("archive format was not recognized")
		}
		return archiveListResult{}, fmt.Errorf("identify archive format: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return archiveListResult{}, fmt.Errorf("rewind archive file: %w", err)
	}

	extractor, ok := format.(archives.Extraction)
	if !ok {
		return archiveListResult{}, fmt.Errorf("archive format does not support listing")
	}

	result := archiveListResult{
		Entries: make([]ArchiveEntry, 0),
	}
	var totalUncompressed int64

	err = extractor.Extract(context.Background(), file, func(_ context.Context, info archives.FileInfo) error {
		entryName, entryErr := cleanArchiveEntryName(info.NameInArchive)
		if entryErr != nil {
			return entryErr
		}
		if archiveEntryIgnored(entryName) {
			return nil
		}
		if archiveEntryDepth(entryName) > MaxArchivePathDepth {
			result.LimitReached = true
			result.LimitReason = fmt.Sprintf("Archive entry depth exceeds the limit of %d.", MaxArchivePathDepth)
			return errArchiveLimitReached
		}
		if info.LinkTarget != "" || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive entry %q is a link", info.NameInArchive)
		}

		sizeBytes := info.Size()
		if !info.IsDir() {
			totalUncompressed += sizeBytes
			if totalUncompressed > MaxArchiveUncompressedSum {
				result.LimitReached = true
				result.LimitReason = fmt.Sprintf(
					"Archive uncompressed size exceeds the limit of %d bytes.",
					MaxArchiveUncompressedSum,
				)
				return errArchiveLimitReached
			}
		}

		if len(result.Entries) >= MaxArchiveFiles {
			result.LimitReached = true
			result.LimitReason = fmt.Sprintf("Archive file count exceeds the limit of %d.", MaxArchiveFiles)
			return errArchiveLimitReached
		}

		result.Entries = append(result.Entries, ArchiveEntry{
			Path:        entryName,
			SizeBytes:   sizeBytes,
			IsDirectory: info.IsDir(),
		})

		return nil
	})
	if err != nil && !errors.Is(err, errArchiveLimitReached) {
		return archiveListResult{}, fmt.Errorf("list archive entries: %w", err)
	}

	return result, nil
}

var errArchiveLimitReached = errors.New("archive listing limit reached")

func cleanArchiveEntryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("archive entry path is empty")
	}
	name = strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(name, "/") || path.IsAbs(name) || filepath.IsAbs(name) || archiveWindowsVolumePath.MatchString(name) {
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

func archiveEntryDepth(name string) int {
	if name == "" {
		return 0
	}

	return strings.Count(name, "/")
}

func archiveEntryIgnored(name string) bool {
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if fileignore.Has(part) {
			return true
		}
	}

	return false
}
