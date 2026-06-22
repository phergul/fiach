package reshade

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	DefaultEffectCatalogueURL = "https://raw.githubusercontent.com/crosire/reshade-shaders/list/EffectPackages.ini"
	DefaultAddonCatalogueURL  = "https://raw.githubusercontent.com/crosire/reshade-shaders/list/Addons.ini"
)

type ContentCatalogueOptions struct {
	EffectCatalogueURL string
	AddonCatalogueURL  string
	HTTPClient         *http.Client
}

type contentCatalogueCache struct {
	EffectContents string    `json:"effectContents"`
	AddonContents  string    `json:"addonContents"`
	FetchedAt      time.Time `json:"fetchedAt"`
}

func ListContentCatalogue(ctx context.Context, dataDir string, refresh bool, options ContentCatalogueOptions) (catalogue ContentCatalogue, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("list ReShade content catalogue: %w", err)
		}
	}()
	if options.EffectCatalogueURL == "" {
		options.EffectCatalogueURL = DefaultEffectCatalogueURL
	}
	if options.AddonCatalogueURL == "" {
		options.AddonCatalogueURL = DefaultAddonCatalogueURL
	}
	if options.HTTPClient == nil {
		options.HTTPClient = http.DefaultClient
	}
	cachePath := filepath.Join(dataDir, "cache", "content-catalogue.json")
	cached, cacheErr := readContentCatalogueCache(cachePath)
	if !refresh && cacheErr == nil {
		catalogue, err = parseContentCatalogue(cached.EffectContents, cached.AddonContents)
		catalogue.Cached = true
		return catalogue, err
	}
	effects, effectErr := fetchText(ctx, options.HTTPClient, options.EffectCatalogueURL)
	addons, addonErr := fetchText(ctx, options.HTTPClient, options.AddonCatalogueURL)
	if effectErr == nil && addonErr == nil {
		catalogue, err = parseContentCatalogue(effects, addons)
		if err != nil {
			return ContentCatalogue{}, err
		}
		if writeErr := writeContentCatalogueCache(cachePath, contentCatalogueCache{
			EffectContents: effects,
			AddonContents:  addons,
			FetchedAt:      time.Now().UTC(),
		}); writeErr != nil {
			return ContentCatalogue{}, writeErr
		}
		return catalogue, nil
	}
	if cacheErr == nil {
		catalogue, err = parseContentCatalogue(cached.EffectContents, cached.AddonContents)
		catalogue.Cached = true
		return catalogue, err
	}
	if effectErr != nil {
		return ContentCatalogue{}, effectErr
	}
	return ContentCatalogue{}, addonErr
}

func fetchText(ctx context.Context, client *http.Client, rawURL string) (contents string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("fetch catalogue %q: %w", rawURL, err)
		}
	}()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "https" || parsed.Host != "raw.githubusercontent.com" {
		return "", errors.New("catalogue URL host is not trusted")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("unexpected status %s", response.Status)
	}
	bytes, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func readContentCatalogueCache(path string) (contentCatalogueCache, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return contentCatalogueCache{}, err
	}
	var cached contentCatalogueCache
	if err := json.Unmarshal(contents, &cached); err != nil {
		return contentCatalogueCache{}, err
	}
	if strings.TrimSpace(cached.EffectContents) == "" || strings.TrimSpace(cached.AddonContents) == "" {
		return contentCatalogueCache{}, errors.New("cached ReShade catalogue is incomplete")
	}
	return cached, nil
}

func writeContentCatalogueCache(path string, cached contentCatalogueCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".catalogue-*.tmp")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if _, err := temp.Write(contents); err != nil {
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

func parseContentCatalogue(effectContents string, addonContents string) (ContentCatalogue, error) {
	effects, err := parseEffectPackages(effectContents)
	if err != nil {
		return ContentCatalogue{}, err
	}
	addons, err := parseAddonPackages(addonContents)
	if err != nil {
		return ContentCatalogue{}, err
	}
	return ContentCatalogue{
		Effects: effects,
		Addons:  addons,
	}, nil
}

func parseEffectPackages(contents string) ([]EffectPackage, error) {
	sections := parseLooseINI(contents)
	result := make([]EffectPackage, 0, len(sections))
	for _, section := range sections {
		item := EffectPackage{
			ID:                 section.id,
			Name:               section.values["PackageName"],
			Description:        section.values["PackageDescription"],
			InstallPath:        section.values["InstallPath"],
			TextureInstallPath: section.values["TextureInstallPath"],
			DownloadURL:        section.values["DownloadUrl"],
			RepositoryURL:      section.values["RepositoryUrl"],
			Required:           section.values["Required"] == "1",
			Enabled:            section.values["Enabled"] == "1",
			EffectFiles:        splitCatalogueList(section.values["EffectFiles"]),
			DenyEffectFiles:    splitCatalogueList(section.values["DenyEffectFiles"]),
		}
		item.Modifiable = !item.Required
		if item.Required {
			item.Enabled = true
		}
		if err := validateEffectPackage(item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i int, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func parseAddonPackages(contents string) ([]AddonPackage, error) {
	sections := parseLooseINI(contents)
	result := make([]AddonPackage, 0, len(sections))
	for _, section := range sections {
		item := AddonPackage{
			ID:                section.id,
			Name:              section.values["PackageName"],
			Description:       section.values["PackageDescription"],
			EffectInstallPath: section.values["EffectInstallPath"],
			DownloadURL:       section.values["DownloadUrl"],
			DownloadURL32:     section.values["DownloadUrl32"],
			DownloadURL64:     section.values["DownloadUrl64"],
			RepositoryURL:     section.values["RepositoryUrl"],
		}
		if !addonPackageHasDownloadURL(item) {
			continue
		}
		if err := validateAddonPackage(item); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	sort.Slice(result, func(i int, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func addonPackageHasDownloadURL(item AddonPackage) bool {
	return strings.TrimSpace(item.DownloadURL) != "" ||
		strings.TrimSpace(item.DownloadURL32) != "" ||
		strings.TrimSpace(item.DownloadURL64) != ""
}

type looseINISection struct {
	id     string
	values map[string]string
}

var looseKeyPattern = regexp.MustCompile(`(?:^|\s)([A-Za-z][A-Za-z0-9]*)=`)

func parseLooseINI(contents string) []looseINISection {
	var result []looseINISection
	current := -1
	for _, line := range strings.Split(strings.ReplaceAll(contents, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		for line != "" {
			if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
				break
			}
			if strings.HasPrefix(line, "[") {
				end := strings.Index(line, "]")
				if end < 0 {
					break
				}
				id := strings.TrimSpace(line[1:end])
				if id != "" {
					result = append(result, looseINISection{
						id:     id,
						values: map[string]string{},
					})
					current = len(result) - 1
				}
				line = strings.TrimSpace(line[end+1:])
				continue
			}
			if current < 0 {
				break
			}
			matches := looseKeyPattern.FindAllStringSubmatchIndex(line, -1)
			if len(matches) == 0 {
				break
			}
			for index, match := range matches {
				key := line[match[2]:match[3]]
				valueStart := match[1]
				valueEnd := len(line)
				if index+1 < len(matches) {
					valueEnd = matches[index+1][0]
				}
				value := strings.TrimSpace(line[valueStart:valueEnd])
				result[current].values[key] = trimInlineComment(value)
			}
			break
		}
	}
	return result
}

func trimInlineComment(value string) string {
	value = strings.TrimSpace(value)
	for _, marker := range []string{" #", " ;"} {
		if index := strings.Index(value, marker); index >= 0 {
			value = strings.TrimSpace(value[:index])
		}
	}
	return value
}

func splitCatalogueList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func validateEffectPackage(item EffectPackage) error {
	if strings.TrimSpace(item.ID) == "" {
		return errors.New("effect package ID is required")
	}
	if strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf("effect package %q name is required", item.ID)
	}
	if strings.TrimSpace(item.DownloadURL) == "" {
		return fmt.Errorf("effect package %q download URL is required", item.ID)
	}
	if err := validateTrustedContentURL(item.DownloadURL); err != nil {
		return fmt.Errorf("effect package %q download URL: %w", item.ID, err)
	}
	if err := validateCatalogueInstallPath(item.InstallPath); err != nil {
		return fmt.Errorf("effect package %q install path: %w", item.ID, err)
	}
	if err := validateCatalogueInstallPath(item.TextureInstallPath); err != nil {
		return fmt.Errorf("effect package %q texture install path: %w", item.ID, err)
	}
	return nil
}

func validateAddonPackage(item AddonPackage) error {
	if strings.TrimSpace(item.ID) == "" {
		return errors.New("add-on package ID is required")
	}
	if strings.TrimSpace(item.Name) == "" {
		return fmt.Errorf("add-on package %q name is required", item.ID)
	}
	if item.DownloadURL == "" && item.DownloadURL32 == "" && item.DownloadURL64 == "" {
		return fmt.Errorf("add-on package %q download URL is required", item.ID)
	}
	for _, rawURL := range []string{item.DownloadURL, item.DownloadURL32, item.DownloadURL64} {
		if rawURL == "" {
			continue
		}
		if err := validateTrustedContentURL(rawURL); err != nil {
			return fmt.Errorf("add-on package %q download URL: %w", item.ID, err)
		}
	}
	if err := validateCatalogueInstallPath(item.EffectInstallPath); err != nil {
		return fmt.Errorf("add-on package %q effect install path: %w", item.ID, err)
	}
	return nil
}

func validateTrustedContentURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "https" {
		return errors.New("URL must use HTTPS")
	}
	switch parsed.Host {
	case "github.com", "raw.githubusercontent.com":
		return nil
	default:
		return fmt.Errorf("host %q is not trusted", parsed.Host)
	}
}

func validateCatalogueInstallPath(value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	clean := filepath.Clean(strings.ReplaceAll(strings.ReplaceAll(value, "\\", string(filepath.Separator)), "/", string(filepath.Separator)))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("path must stay inside the target")
	}
	return nil
}
