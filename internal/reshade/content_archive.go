package reshade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/mholt/archives"

	"github.com/phergul/fiach/internal/fileops"
)

type contentArchiveFile struct {
	Path      string
	SHA256    string
	SizeBytes int64
}

func ensureContentArchive(ctx context.Context, dataDir string, rawURL string) (path string, sha string, size int64, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("cache ReShade content archive: %w", err)
		}
	}()
	if err := validateTrustedContentURL(rawURL); err != nil {
		return "", "", 0, err
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", 0, err
	}
	name := filepath.Base(parsed.Path)
	if strings.TrimSpace(name) == "" || name == "." || name == "/" {
		name = "download.bin"
	}
	cacheName := hashBytes([]byte(rawURL))[:16] + "-" + filetxnSafeSegment(name)
	cachePath := filepath.Join(dataDir, "cache", "content-archives", cacheName)
	if hash, cachedSize, matchErr := fileops.FileIntegrity(cachePath); matchErr == nil {
		return cachePath, hash, cachedSize, nil
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return "", "", 0, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", 0, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", "", 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", "", 0, fmt.Errorf("download %q: unexpected status %s", rawURL, response.Status)
	}
	temp, err := os.CreateTemp(filepath.Dir(cachePath), ".content-*.tmp")
	if err != nil {
		return "", "", 0, err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := io.Copy(temp, response.Body); err != nil {
		_ = temp.Close()
		return "", "", 0, err
	}
	if err := temp.Close(); err != nil {
		return "", "", 0, err
	}
	if err := os.Remove(cachePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", 0, err
	}
	if err := os.Rename(tempPath, cachePath); err != nil {
		return "", "", 0, err
	}
	hash, cachedSize, err := fileops.FileIntegrity(cachePath)
	if err != nil {
		return "", "", 0, err
	}
	return cachePath, hash, cachedSize, nil
}

func extractContentArchive(ctx context.Context, archivePath string, destination string) (files []contentArchiveFile, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("extract ReShade content archive: %w", err)
		}
	}()
	if err := os.RemoveAll(destination); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return nil, err
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	format, _, err := archives.Identify(ctx, filepath.Base(archivePath), file)
	if err != nil {
		return nil, fmt.Errorf("identify archive format: %w", err)
	}
	extractor, ok := format.(archives.Extraction)
	if !ok {
		return nil, errors.New("content archive format is not extractable")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	err = extractor.Extract(ctx, file, func(ctx context.Context, info archives.FileInfo) error {
		name, ignored, err := cleanContentArchiveEntry(info)
		if err != nil || ignored {
			return err
		}
		key := strings.ToLower(name)
		if seen[key] {
			return fmt.Errorf("archive contains a case-insensitive duplicate path %q", name)
		}
		seen[key] = true
		target, err := contentArchiveDestination(destination, name)
		if err != nil {
			return err
		}
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
		if _, err := io.Copy(output, source); err != nil {
			_ = output.Close()
			return err
		}
		if err := output.Close(); err != nil {
			return err
		}
		hash, size, err := fileops.FileIntegrity(target)
		if err != nil {
			return err
		}
		files = append(files, contentArchiveFile{
			Path:      target,
			SHA256:    hash,
			SizeBytes: size,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(files) == 0 {
		return nil, errors.New("content archive contains no files")
	}
	return files, nil
}

func cleanContentArchiveEntry(info archives.FileInfo) (string, bool, error) {
	name := strings.TrimSpace(strings.ReplaceAll(info.NameInArchive, "\\", "/"))
	if name == "" {
		return "", false, errors.New("archive entry path is empty")
	}
	if strings.HasPrefix(name, "/") || path.IsAbs(name) || filepath.IsAbs(name) {
		return "", false, fmt.Errorf("archive entry %q is absolute", name)
	}
	clean := path.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.Contains(clean, "/../") {
		return "", false, fmt.Errorf("archive entry %q escapes the archive root", name)
	}
	if slices.Contains(strings.Split(clean, "/"), "__MACOSX") {
		return clean, true, nil
	}
	mode := info.Mode()
	if mode&os.ModeSymlink != 0 || info.LinkTarget != "" {
		return "", false, fmt.Errorf("archive entry %q is a link", name)
	}
	if !info.IsDir() && !mode.IsRegular() {
		return "", false, fmt.Errorf("archive entry %q is a special file", name)
	}
	return clean, false, nil
}

func contentArchiveDestination(root string, entry string) (string, error) {
	target := filepath.Join(root, filepath.FromSlash(entry))
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return "", err
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry %q escapes staging", entry)
	}
	return target, nil
}

func filetxnSafeSegment(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_' {
			builder.WriteRune(r)
		} else {
			builder.WriteByte('_')
		}
	}
	clean := strings.Trim(builder.String(), "._-")
	if clean == "" {
		return "file"
	}
	return clean
}

func contentHash(values ...string) string {
	hash := sha256.New()
	for _, value := range values {
		_, _ = hash.Write([]byte(value))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
