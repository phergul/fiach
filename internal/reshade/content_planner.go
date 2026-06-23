package reshade

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage/dbtypes"
)

func (m *Manager) planContent(
	ctx context.Context,
	gameRoot string,
	targetPath string,
	request Request,
	existing dbtypes.ReShadeTarget,
	manifest Manifest,
) (Preview, error) {
	if err := requireExistingLayout(request, &existing); err != nil {
		return Preview{}, err
	}
	catalogue, err := ListContentCatalogue(ctx, m.dataDir, false, ContentCatalogueOptions{})
	if err != nil {
		return Preview{}, err
	}
	effectsByID := mapEffectPackages(catalogue.Effects)
	addonsByID := mapAddonPackages(catalogue.Addons)
	selections := normalizeContentSelections(request.Content, catalogue.Effects)
	selectedSources := map[string]bool{}
	operations := []Operation{}
	impacts := []PathImpact{}
	warnings := []string{}
	conflicts := []string{}
	desiredFiles := nonContentManifestFiles(manifest.Files)
	plannedByPath := map[string]ManagedFile{}
	existingByPath := manifestFilesByPath(manifest.Files)
	configEffectPaths := []string{}
	configTexturePaths := []string{}
	configAddonPath := ""

	for _, selection := range selections.EffectPackages {
		pkg, ok := effectsByID[selection.ID]
		if !ok {
			conflicts = append(conflicts, fmt.Sprintf("Selected ReShade effect package %q is not in the catalogue.", selection.ID))
			continue
		}
		sourceKey := contentSourceKey(ContentSourceEffectPackage, pkg.ID)
		selectedSources[sourceKey] = true
		packageFiles, archiveSHA, archiveSize, err := m.stageEffectPackage(ctx, targetPath, pkg, selection)
		if err != nil {
			return Preview{}, err
		}
		for _, file := range packageFiles {
			source := ContentSource{
				Kind:          ContentSourceEffectPackage,
				ID:            pkg.ID,
				Name:          pkg.Name,
				RepositoryURL: pkg.RepositoryURL,
				DownloadURL:   pkg.DownloadURL,
				ArchiveSHA256: archiveSHA,
				ArchiveSize:   archiveSize,
			}
			managed := ManagedFile{
				RelativePath: file.relativePath,
				SHA256:       file.sha256,
				SizeBytes:    file.sizeBytes,
				Ownership:    OwnershipManaged,
				Role:         file.role,
				Sources:      []ContentSource{source},
			}
			merged, add, conflict := mergePlannedContentFile(targetPath, existingByPath, plannedByPath, managed)
			if conflict != "" {
				conflicts = append(conflicts, conflict)
				continue
			}
			if !add {
				continue
			}
			plannedByPath[strings.ToLower(filepath.Clean(file.relativePath))] = merged
			if file.role == PathRoleEffects && strings.EqualFold(filepath.Ext(file.relativePath), ".fx") {
				if duplicate := duplicateEffectConflict(targetPath, file.relativePath); duplicate != "" {
					conflicts = append(conflicts, duplicate)
				}
			}
			operations = append(operations, Operation{
				Type:       "copy",
				SourcePath: file.stagedPath,
				TargetPath: filepath.Join(targetPath, file.relativePath),
				SHA256:     file.sha256,
				SizeBytes:  file.sizeBytes,
			})
			impacts = append(impacts, pathImpact(
				file.relativePath,
				file.role,
				"install",
				OwnershipManaged,
				pathExists(filepath.Join(targetPath, file.relativePath)),
				false,
			))
		}
		if pkg.InstallPath != "" {
			configEffectPaths = append(configEffectPaths, relativeINIPath(pkg.InstallPath))
		}
		if pkg.TextureInstallPath != "" {
			configTexturePaths = append(configTexturePaths, relativeINIPath(pkg.TextureInstallPath))
		}
	}

	if len(request.Content.Addons) > 0 {
		if existing.BuildVariant != string(BuildVariantAddon) {
			conflicts = append(conflicts, "ReShade add-ons require the full add-on build.")
		}
		warnings = append(warnings,
			"ReShade add-ons use the unsigned full add-on build and are intended for single-player use.",
			"Anti-cheat protected games may ban or block add-on usage.",
		)
	}
	for _, selection := range request.Content.Addons {
		addon, ok := addonsByID[selection.ID]
		if !ok {
			conflicts = append(conflicts, fmt.Sprintf("Selected ReShade add-on %q is not in the catalogue.", selection.ID))
			continue
		}
		sourceKey := contentSourceKey(ContentSourceAddon, addon.ID)
		selectedSources[sourceKey] = true
		file, archiveSHA, archiveSize, err := m.stageAddon(ctx, existing, addon)
		if err != nil {
			return Preview{}, err
		}
		source := ContentSource{
			Kind:          ContentSourceAddon,
			ID:            addon.ID,
			Name:          addon.Name,
			RepositoryURL: addon.RepositoryURL,
			DownloadURL:   addonDownloadURL(addon, Architecture(existing.Architecture)),
			ArchiveSHA256: archiveSHA,
			ArchiveSize:   archiveSize,
		}
		managed := ManagedFile{
			RelativePath: file.relativePath,
			SHA256:       file.sha256,
			SizeBytes:    file.sizeBytes,
			Ownership:    OwnershipManaged,
			Role:         PathRoleAddons,
			Sources:      []ContentSource{source},
		}
		merged, add, conflict := mergePlannedContentFile(targetPath, existingByPath, plannedByPath, managed)
		if conflict != "" {
			conflicts = append(conflicts, conflict)
			continue
		}
		if !add {
			continue
		}
		plannedByPath[strings.ToLower(filepath.Clean(file.relativePath))] = merged
		operations = append(operations, Operation{
			Type:       "copy",
			SourcePath: file.stagedPath,
			TargetPath: filepath.Join(targetPath, file.relativePath),
			SHA256:     file.sha256,
			SizeBytes:  file.sizeBytes,
		})
		impacts = append(impacts, pathImpact(
			file.relativePath,
			PathRoleAddons,
			"install",
			OwnershipManaged,
			pathExists(filepath.Join(targetPath, file.relativePath)),
			false,
		))
		configAddonPath = "Addons"
		if addon.EffectInstallPath != "" {
			configEffectPaths = append(configEffectPaths, relativeINIPath(addon.EffectInstallPath))
		}
	}

	for _, file := range manifest.Files {
		if !isContentManagedFile(file) {
			continue
		}
		if contentFileRetained(file, selectedSources) {
			continue
		}
		target := filepath.Join(targetPath, file.RelativePath)
		operations = append(operations, Operation{
			Type:       "delete",
			TargetPath: target,
		})
		impacts = append(impacts, pathImpact(
			file.RelativePath,
			file.Role,
			"remove",
			OwnershipManaged,
			pathExists(target),
			false,
		))
	}
	for _, file := range plannedByPath {
		desiredFiles = append(desiredFiles, file)
	}
	configOperations, configImpacts, configFile, err := m.planContentConfigUpdate(
		targetPath,
		manifest,
		configEffectPaths,
		configTexturePaths,
		configAddonPath,
	)
	if err != nil {
		return Preview{}, err
	}
	operations = append(operations, configOperations...)
	impacts = append(impacts, configImpacts...)
	if configFile != nil {
		desiredFiles = replaceOrAppendManifestFile(desiredFiles, *configFile)
	}
	sortManagedFiles(desiredFiles)
	manifest.Files = desiredFiles
	return Preview{
		Operations:       operations,
		PathImpacts:      impacts,
		Warnings:         warnings,
		Conflicts:        conflicts,
		Drift:            []Drift{},
		UserContentDrift: []UserContentDrift{},
		DesiredTarget: &TargetState{
			RuntimeVersion:   existing.RuntimeVersion,
			Provenance:       provenanceFromRow(&existing),
			ManagementOrigin: existing.ManagementOrigin,
			Manifest:         manifest,
		},
	}, nil
}

type stagedContentFile struct {
	stagedPath   string
	relativePath string
	role         PathRole
	sha256       string
	sizeBytes    int64
}

func (m *Manager) stageEffectPackage(ctx context.Context, targetPath string, pkg EffectPackage, selection EffectPackageSelection) ([]stagedContentFile, string, int64, error) {
	archivePath, archiveSHA, archiveSize, err := ensureContentArchive(ctx, m.dataDir, pkg.DownloadURL)
	if err != nil {
		return nil, "", 0, err
	}
	stagingRoot := filepath.Join(m.dataDir, "staging", "content", contentHash(pkg.ID, archiveSHA))
	files, err := extractContentArchive(ctx, archivePath, stagingRoot)
	if err != nil {
		return nil, "", 0, err
	}
	shaderRoot, textureRoot := inferEffectPackageRoots(files)
	selectedEffects := map[string]bool{}
	for _, file := range selection.EffectFiles {
		selectedEffects[strings.ToLower(filepath.Base(file))] = true
	}
	deny := map[string]bool{}
	for _, file := range pkg.DenyEffectFiles {
		deny[strings.ToLower(filepath.Base(file))] = true
	}
	var result []stagedContentFile
	for _, file := range files {
		role := PathRole("")
		base := strings.ToLower(filepath.Base(file.Path))
		sourceRoot := ""
		targetRoot := ""
		if shaderRoot != "" && pathIsWithin(file.Path, shaderRoot) {
			role = PathRoleEffects
			sourceRoot = shaderRoot
			targetRoot = pkg.InstallPath
			if deny[base] {
				continue
			}
			if len(selectedEffects) > 0 && strings.EqualFold(filepath.Ext(file.Path), ".fx") && !selectedEffects[base] {
				continue
			}
		} else if textureRoot != "" && pathIsWithin(file.Path, textureRoot) {
			role = PathRoleTextures
			sourceRoot = textureRoot
			targetRoot = pkg.TextureInstallPath
		}
		if role == "" {
			continue
		}
		relative, err := filepath.Rel(sourceRoot, file.Path)
		if err != nil {
			return nil, "", 0, err
		}
		targetRelative, err := contentTargetRelativePath(targetRoot, relative)
		if err != nil {
			return nil, "", 0, err
		}
		if _, err := ResolveWithinRoot(targetPath, targetRelative); err != nil {
			return nil, "", 0, err
		}
		result = append(result, stagedContentFile{
			stagedPath:   file.Path,
			relativePath: targetRelative,
			role:         role,
			sha256:       file.SHA256,
			sizeBytes:    file.SizeBytes,
		})
	}
	if len(result) == 0 {
		return nil, "", 0, fmt.Errorf("effect package %q produced no installable files", pkg.ID)
	}
	return result, archiveSHA, archiveSize, nil
}

func (m *Manager) stageAddon(ctx context.Context, existing dbtypes.ReShadeTarget, addon AddonPackage) (stagedContentFile, string, int64, error) {
	rawURL := addonDownloadURL(addon, Architecture(existing.Architecture))
	if rawURL == "" {
		return stagedContentFile{}, "", 0, fmt.Errorf("add-on %q has no download URL for %s", addon.ID, existing.Architecture)
	}
	downloadPath, archiveSHA, archiveSize, err := ensureContentArchive(ctx, m.dataDir, rawURL)
	if err != nil {
		return stagedContentFile{}, "", 0, err
	}
	if isAddonBinary(downloadPath) {
		targetName := addonTargetName(downloadPath, Architecture(existing.Architecture))
		return stagedContentFile{
			stagedPath:   downloadPath,
			relativePath: filepath.Join("Addons", targetName),
			role:         PathRoleAddons,
			sha256:       archiveSHA,
			sizeBytes:    archiveSize,
		}, archiveSHA, archiveSize, nil
	}
	stagingRoot := filepath.Join(m.dataDir, "staging", "addons", contentHash(addon.ID, archiveSHA))
	files, err := extractContentArchive(ctx, downloadPath, stagingRoot)
	if err != nil {
		return stagedContentFile{}, "", 0, err
	}
	for _, file := range files {
		if !addonBinaryMatches(file.Path, Architecture(existing.Architecture)) {
			continue
		}
		return stagedContentFile{
			stagedPath:   file.Path,
			relativePath: filepath.Join("Addons", addonTargetName(file.Path, Architecture(existing.Architecture))),
			role:         PathRoleAddons,
			sha256:       file.SHA256,
			sizeBytes:    file.SizeBytes,
		}, archiveSHA, archiveSize, nil
	}
	return stagedContentFile{}, "", 0, fmt.Errorf("add-on %q archive contains no %s add-on binary", addon.ID, existing.Architecture)
}

func (m *Manager) planContentConfigUpdate(
	targetPath string,
	manifest Manifest,
	effectPaths []string,
	texturePaths []string,
	addonPath string,
) ([]Operation, []PathImpact, *ManagedFile, error) {
	if len(effectPaths) == 0 && len(texturePaths) == 0 && addonPath == "" {
		return nil, nil, nil, nil
	}
	configPath := filepath.Join(targetPath, "ReShade.ini")
	existingPaths, _, err := parseReShadeContentPaths(configPath)
	if err != nil {
		return nil, nil, nil, err
	}
	effects := mergeSearchPaths(existingPaths, PathRoleEffects, effectPaths)
	textures := mergeSearchPaths(existingPaths, PathRoleTextures, texturePaths)
	staged, hash, size, err := stageUpdatedReShadeINI(m.dataDir, configPath, effects, textures, addonPath)
	if err != nil {
		return nil, nil, nil, err
	}
	file := ManagedFile{
		RelativePath: "ReShade.ini",
		SHA256:       hash,
		SizeBytes:    size,
		Ownership:    OwnershipUser,
		Role:         PathRoleConfiguration,
	}
	for _, existing := range manifest.Files {
		if strings.EqualFold(filepath.Clean(existing.RelativePath), "ReShade.ini") && existing.BackupPath != nil {
			file.BackupPath = existing.BackupPath
			file.BackupSHA256 = existing.BackupSHA256
			file.BackupSize = existing.BackupSize
		}
	}
	return []Operation{{
			Type:       "copy",
			SourcePath: staged,
			TargetPath: configPath,
			SHA256:     hash,
			SizeBytes:  size,
		}}, []PathImpact{pathImpact(
			"ReShade.ini",
			PathRoleConfiguration,
			"update search paths",
			OwnershipUser,
			pathExists(configPath),
			false,
		)}, &file, nil
}

func normalizeContentSelections(request ContentRequest, effects []EffectPackage) ContentRequest {
	seenEffects := map[string]EffectPackageSelection{}
	for _, selection := range request.EffectPackages {
		if strings.TrimSpace(selection.ID) == "" {
			continue
		}
		selection.EffectFiles = deduplicateStrings(selection.EffectFiles)
		seenEffects[selection.ID] = selection
	}
	for _, pkg := range effects {
		if pkg.Required {
			seenEffects[pkg.ID] = EffectPackageSelection{ID: pkg.ID}
		}
	}
	result := ContentRequest{}
	for _, selection := range seenEffects {
		result.EffectPackages = append(result.EffectPackages, selection)
	}
	slices.SortFunc(result.EffectPackages, func(a EffectPackageSelection, b EffectPackageSelection) int {
		return strings.Compare(a.ID, b.ID)
	})
	seenAddons := map[string]bool{}
	for _, selection := range request.Addons {
		if strings.TrimSpace(selection.ID) == "" || seenAddons[selection.ID] {
			continue
		}
		seenAddons[selection.ID] = true
		result.Addons = append(result.Addons, selection)
	}
	slices.SortFunc(result.Addons, func(a AddonSelection, b AddonSelection) int {
		return strings.Compare(a.ID, b.ID)
	})
	return result
}

func mapEffectPackages(items []EffectPackage) map[string]EffectPackage {
	result := make(map[string]EffectPackage, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func mapAddonPackages(items []AddonPackage) map[string]AddonPackage {
	result := make(map[string]AddonPackage, len(items))
	for _, item := range items {
		result[item.ID] = item
	}
	return result
}

func nonContentManifestFiles(files []ManagedFile) []ManagedFile {
	result := make([]ManagedFile, 0, len(files))
	for _, file := range files {
		if !isContentManagedFile(file) {
			result = append(result, file)
		}
	}
	return result
}

func manifestFilesByPath(files []ManagedFile) map[string]ManagedFile {
	result := make(map[string]ManagedFile, len(files))
	for _, file := range files {
		result[strings.ToLower(filepath.Clean(file.RelativePath))] = file
	}
	return result
}

func isContentManagedFile(file ManagedFile) bool {
	for _, source := range fileSources(file) {
		if source.Kind == ContentSourceEffectPackage || source.Kind == ContentSourceAddon {
			return true
		}
	}
	return false
}

func contentFileRetained(file ManagedFile, selected map[string]bool) bool {
	for _, source := range fileSources(file) {
		if selected[contentSourceKey(source.Kind, source.ID)] {
			return true
		}
	}
	return false
}

func fileSources(file ManagedFile) []ContentSource {
	if len(file.Sources) > 0 {
		return file.Sources
	}
	if file.Source != nil {
		return []ContentSource{*file.Source}
	}
	return nil
}

func contentSourceKey(kind ContentSourceKind, id string) string {
	return string(kind) + ":" + id
}

func mergePlannedContentFile(
	targetPath string,
	existingByPath map[string]ManagedFile,
	plannedByPath map[string]ManagedFile,
	file ManagedFile,
) (ManagedFile, bool, string) {
	key := strings.ToLower(filepath.Clean(file.RelativePath))
	if existing, ok := plannedByPath[key]; ok {
		if existing.SHA256 != file.SHA256 || existing.SizeBytes != file.SizeBytes {
			return ManagedFile{}, false, fmt.Sprintf("Selected packages write different content to %q.", file.RelativePath)
		}
		existing.Sources = append(existing.Sources, file.Sources...)
		for index := range existing.Sources {
			existing.Sources[index].Shared = true
		}
		plannedByPath[key] = existing
		return ManagedFile{}, false, ""
	}
	if existing, ok := existingByPath[key]; ok {
		if isContentManagedFile(existing) {
			if existing.SHA256 == file.SHA256 && existing.SizeBytes == file.SizeBytes {
				file.Sources = append(fileSources(existing), file.Sources...)
				for index := range file.Sources {
					file.Sources[index].Shared = len(file.Sources) > 1
				}
				return file, true, ""
			}
			return ManagedFile{}, false, fmt.Sprintf("Managed content file %q would be replaced by different package content.", file.RelativePath)
		}
	}
	if _, err := os.Stat(filepath.Join(targetPath, file.RelativePath)); err == nil {
		return ManagedFile{}, false, fmt.Sprintf("Existing user-owned ReShade content file %q blocks package installation.", file.RelativePath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return ManagedFile{}, false, fmt.Sprintf("Existing ReShade content file %q cannot be inspected.", file.RelativePath)
	}
	return file, true, ""
}

func inferEffectPackageRoots(files []contentArchiveFile) (string, string) {
	shaderRoot := firstNamedDirectory(files, "Shaders")
	textureRoot := firstNamedDirectory(files, "Textures")
	if shaderRoot == "" {
		shaderRoot = shortestParentWithExtension(files, ".fx")
	}
	if textureRoot == "" {
		textureRoot = shortestParentWithAnyExtension(files, []string{".png", ".jpg", ".jpeg", ".dds"})
	}
	return shaderRoot, textureRoot
}

func firstNamedDirectory(files []contentArchiveFile, name string) string {
	for _, file := range files {
		dir := filepath.Dir(file.Path)
		for {
			if strings.EqualFold(filepath.Base(dir), name) {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	return ""
}

func shortestParentWithExtension(files []contentArchiveFile, extension string) string {
	return shortestParentWithAnyExtension(files, []string{extension})
}

func shortestParentWithAnyExtension(files []contentArchiveFile, extensions []string) string {
	best := ""
	for _, file := range files {
		if !slices.ContainsFunc(extensions, func(extension string) bool {
			return strings.EqualFold(filepath.Ext(file.Path), extension)
		}) {
			continue
		}
		dir := filepath.Dir(file.Path)
		if best == "" || len(dir) < len(best) {
			best = dir
		}
	}
	return best
}

func pathIsWithin(path string, root string) bool {
	return fileops.RequirePathWithinRoot("content path", path, root) == nil
}

func contentTargetRelativePath(root string, child string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		root = "."
	}
	cleanRoot := filepath.Clean(strings.ReplaceAll(strings.ReplaceAll(root, "\\", string(filepath.Separator)), "/", string(filepath.Separator)))
	clean := filepath.Clean(filepath.Join(cleanRoot, child))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("content target path %q escapes target", clean)
	}
	return clean, nil
}

func duplicateEffectConflict(targetPath string, relativePath string) string {
	configPath := filepath.Join(targetPath, "ReShade.ini")
	paths, _, err := parseReShadeContentPaths(configPath)
	if err != nil {
		return ""
	}
	targetDir := filepath.Dir(filepath.Join(targetPath, relativePath))
	filename := filepath.Base(relativePath)
	for _, item := range paths {
		if item.Role != PathRoleEffects {
			continue
		}
		resolved := strings.ReplaceAll(item.Value, "\\", string(filepath.Separator))
		wildcard := strings.HasSuffix(resolved, "**")
		resolved = strings.TrimSuffix(strings.TrimSuffix(resolved, "**"), string(filepath.Separator))
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(targetPath, resolved)
		}
		if strings.EqualFold(filepath.Clean(resolved), filepath.Clean(targetDir)) {
			continue
		}
		candidate := filepath.Join(resolved, filename)
		if wildcard {
			found := ""
			_ = filepath.WalkDir(resolved, func(path string, entry os.DirEntry, err error) error {
				if err != nil || entry.IsDir() || !strings.EqualFold(entry.Name(), filename) {
					return nil
				}
				found = path
				return filepath.SkipAll
			})
			candidate = found
		}
		if candidate != "" && pathExists(candidate) {
			return fmt.Sprintf("Duplicate ReShade effect %q exists in non-default search path %q.", filename, filepath.Dir(candidate))
		}
	}
	return ""
}

func addonDownloadURL(addon AddonPackage, architecture Architecture) string {
	if architecture == ArchitectureX86 && addon.DownloadURL32 != "" {
		return addon.DownloadURL32
	}
	if architecture == ArchitectureX64 && addon.DownloadURL64 != "" {
		return addon.DownloadURL64
	}
	return addon.DownloadURL
}

func isAddonBinary(path string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	return extension == ".addon" || extension == ".addon32" || extension == ".addon64"
}

func addonBinaryMatches(path string, architecture Architecture) bool {
	extension := strings.ToLower(filepath.Ext(path))
	if extension == ".addon" {
		return true
	}
	if architecture == ArchitectureX86 {
		return extension == ".addon32"
	}
	return extension == ".addon64"
}

func addonTargetName(path string, architecture Architecture) string {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if architecture == ArchitectureX86 {
		return name + ".addon32"
	}
	return name + ".addon64"
}

func replaceOrAppendManifestFile(files []ManagedFile, replacement ManagedFile) []ManagedFile {
	for index := range files {
		if strings.EqualFold(filepath.Clean(files[index].RelativePath), filepath.Clean(replacement.RelativePath)) {
			files[index] = replacement
			return files
		}
	}
	return append(files, replacement)
}

func sortManagedFiles(files []ManagedFile) {
	slices.SortFunc(files, func(a ManagedFile, b ManagedFile) int {
		return strings.Compare(
			strings.ToLower(filepath.Clean(a.RelativePath)),
			strings.ToLower(filepath.Clean(b.RelativePath)),
		)
	})
}
