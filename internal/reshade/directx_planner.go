package reshade

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/storage/dbtypes"
	"github.com/phergul/fiach/internal/winversion"
)

type DirectXPlannerOptions struct {
	ResolveInstaller    func(context.Context, InstallerVariant, InstallerResolveOptions) (InstallerRelease, error)
	AcquireInstaller    func(context.Context, InstallerRelease, InstallerAcquireOptions) (InstallerArtifact, error)
	PrepareSetup        func(context.Context, SetupRequest, SetupRunnerOptions) (SetupRunResult, error)
	InspectArchitecture func(string) (Architecture, error)
	ReadMetadata        func(string) (winversion.Metadata, error)
}

type directXPlanner struct {
	resolveInstaller    func(context.Context, InstallerVariant, InstallerResolveOptions) (InstallerRelease, error)
	acquireInstaller    func(context.Context, InstallerRelease, InstallerAcquireOptions) (InstallerArtifact, error)
	prepareSetup        func(context.Context, SetupRequest, SetupRunnerOptions) (SetupRunResult, error)
	inspectArchitecture func(string) (Architecture, error)
	readMetadata        func(string) (winversion.Metadata, error)
}

func NewDirectXPlanner(options DirectXPlannerOptions) Planner {
	if options.ResolveInstaller == nil {
		options.ResolveInstaller = ResolveLatestInstaller
	}
	if options.AcquireInstaller == nil {
		options.AcquireInstaller = AcquireInstaller
	}
	if options.PrepareSetup == nil {
		options.PrepareSetup = PrepareSetup
	}
	if options.InspectArchitecture == nil {
		options.InspectArchitecture = inspectPEArchitecture
	}
	if options.ReadMetadata == nil {
		options.ReadMetadata = winversion.Read
	}
	return &directXPlanner{
		resolveInstaller:    options.ResolveInstaller,
		acquireInstaller:    options.AcquireInstaller,
		prepareSetup:        options.PrepareSetup,
		inspectArchitecture: options.InspectArchitecture,
		readMetadata:        options.ReadMetadata,
	}
}

func (p *directXPlanner) Plan(
	ctx context.Context,
	gameRoot string,
	request Request,
	existing *dbtypes.ReShadeTarget,
) (preview Preview, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("plan managed ReShade DirectX lifecycle: %w", err)
		}
	}()
	targetPath, err := ResolveWithinRoot(gameRoot, request.TargetRelativePath)
	if err != nil {
		return Preview{}, err
	}
	executablePath, err := ResolveWithinRoot(gameRoot, request.ExecutableRelativePath)
	if err != nil {
		return Preview{}, err
	}
	architecture, err := p.inspectArchitecture(executablePath)
	if err != nil {
		return Preview{}, fmt.Errorf("inspect selected executable architecture: %w", err)
	}
	if architecture != request.Architecture {
		return Preview{}, fmt.Errorf(
			"selected executable architecture %q does not match requested architecture %q",
			architecture,
			request.Architecture,
		)
	}
	preview = Preview{
		Operations:       []Operation{},
		PathImpacts:      []PathImpact{},
		Warnings:         []string{},
		Conflicts:        []string{},
		Drift:            []Drift{},
		UserContentDrift: []UserContentDrift{},
	}
	conflicts, conflictImpacts := p.proxyConflicts(targetPath, request, existing)
	if len(conflicts) > 0 {
		preview.Conflicts = append(preview.Conflicts, conflicts...)
		preview.PathImpacts = append(preview.PathImpacts, conflictImpacts...)
		return preview, nil
	}

	switch request.Action {
	case ActionInstall:
		return p.planInstallOrUpdate(ctx, gameRoot, targetPath, executablePath, request, nil, false)
	case ActionAdopt:
		return p.planAdopt(gameRoot, targetPath, request)
	case ActionUpdate:
		return p.planInstallOrUpdate(ctx, gameRoot, targetPath, executablePath, request, existing, true)
	case ActionRepair:
		return p.planRepair(ctx, gameRoot, targetPath, executablePath, request, existing)
	case ActionUninstall:
		return p.planUninstall(gameRoot, targetPath, request, existing)
	default:
		return Preview{}, fmt.Errorf("action %q is unsupported", request.Action)
	}
}

func (p *directXPlanner) planInstallOrUpdate(
	ctx context.Context,
	gameRoot string,
	targetPath string,
	executablePath string,
	request Request,
	existing *dbtypes.ReShadeTarget,
	update bool,
) (Preview, error) {
	variant := installerVariant(request.BuildVariant)
	acknowledgements := installerAcknowledgements(request)
	release, err := p.resolveInstaller(ctx, variant, InstallerResolveOptions{})
	if err != nil {
		return Preview{}, err
	}
	artifact, err := p.acquireInstaller(ctx, release, InstallerAcquireOptions{
		Acknowledgements: acknowledgements,
	})
	if err != nil {
		return Preview{}, err
	}
	setupOperation := SetupOperationInstall
	var existingInputs []SetupInput
	if update && existing != nil {
		if existing.RenderingAPI == string(request.RenderingAPI) &&
			strings.EqualFold(existing.ProxyFilename, request.ProxyFilename) {
			setupOperation = SetupOperationUpdate
		}
		existingInputs, err = setupExistingInputs(targetPath, existing.ProxyFilename)
		if err != nil {
			return Preview{}, err
		}
	}
	setupResult, err := p.prepareSetup(ctx, SetupRequest{
		Artifact:         artifact,
		TargetExecutable: executablePath,
		RenderingAPI:     request.RenderingAPI,
		Operation:        setupOperation,
		Architecture:     request.Architecture,
		ExpectedProxy:    request.ProxyFilename,
		ExistingInputs:   existingInputs,
		ExpectedOutputRelativePaths: []string{
			request.ProxyFilename,
			"ReShade.ini",
			"ReShade.log",
			"ReShadePreset.ini",
		},
		Acknowledgements: acknowledgements,
	}, SetupRunnerOptions{})
	if err != nil {
		return Preview{}, err
	}
	if setupResult.Prepared == nil {
		return Preview{}, errors.New("official ReShade setup produced no prepared output")
	}
	preparedByPath := preparedFilesByRelativePath(setupResult.Prepared.Files)
	proxy, ok := preparedByPath[strings.ToLower(filepath.Clean(request.ProxyFilename))]
	if !ok {
		return Preview{}, errors.New("prepared ReShade runtime proxy is missing")
	}
	operations := []Operation{}
	impacts := []PathImpact{}
	manifestFiles := []ManagedFile{}
	if update && existing != nil && !strings.EqualFold(existing.ProxyFilename, request.ProxyFilename) {
		oldPath := filepath.Join(targetPath, existing.ProxyFilename)
		oldOperation := Operation{
			Type:       "delete",
			TargetPath: oldPath,
		}
		oldAction := "remove"
		oldManifest, decodeErr := DecodeManifest(existing.ManifestJSON)
		if decodeErr != nil {
			return Preview{}, decodeErr
		}
		if oldRuntime, found := runtimeManifestFile(oldManifest, existing.ProxyFilename); found &&
			oldRuntime.BackupPath != nil &&
			oldRuntime.BackupSHA256 != nil &&
			oldRuntime.BackupSize != nil {
			matches, matchErr := fileops.FileMatchesIntegrity(
				*oldRuntime.BackupPath,
				*oldRuntime.BackupSHA256,
				*oldRuntime.BackupSize,
			)
			if matchErr == nil && matches {
				oldOperation = Operation{
					Type:       "restore",
					SourcePath: *oldRuntime.BackupPath,
					TargetPath: oldPath,
					SHA256:     *oldRuntime.BackupSHA256,
					SizeBytes:  *oldRuntime.BackupSize,
				}
				oldAction = "restore backup"
			}
		}
		operations = append(operations, oldOperation)
		impacts = append(impacts, pathImpact(
			existing.ProxyFilename,
			PathRoleRuntime,
			oldAction,
			OwnershipManaged,
			pathExists(oldPath),
			false,
		))
	}
	proxyTarget := filepath.Join(targetPath, request.ProxyFilename)
	operations = append(operations, Operation{
		Type:       "copy",
		SourcePath: proxy.Path,
		TargetPath: proxyTarget,
		SHA256:     proxy.SHA256,
		SizeBytes:  proxy.SizeBytes,
	})
	manifestFiles = append(manifestFiles, managedFileFromPrepared(proxy, OwnershipManaged))
	impacts = append(impacts, pathImpact(
		request.ProxyFilename,
		PathRoleRuntime,
		"replace",
		OwnershipManaged,
		pathExists(proxyTarget),
		false,
	))

	if !update {
		for _, defaultFile := range []struct {
			Name string
			Role PathRole
		}{
			{
				Name: "ReShade.ini",
				Role: PathRoleConfiguration,
			},
			{
				Name: "ReShadePreset.ini",
				Role: PathRolePreset,
			},
		} {
			prepared, found := preparedByPath[strings.ToLower(defaultFile.Name)]
			target := filepath.Join(targetPath, defaultFile.Name)
			if !found || pathExists(target) {
				impacts = append(impacts, pathImpact(
					defaultFile.Name,
					defaultFile.Role,
					"preserve",
					OwnershipUser,
					pathExists(target),
					true,
				))
				continue
			}
			operations = append(operations, Operation{
				Type:       "copy",
				SourcePath: prepared.Path,
				TargetPath: target,
				SHA256:     prepared.SHA256,
				SizeBytes:  prepared.SizeBytes,
			})
			manifestFiles = append(manifestFiles, managedFileFromPrepared(prepared, OwnershipUser))
			impacts = append(impacts, pathImpact(
				defaultFile.Name,
				defaultFile.Role,
				"create",
				OwnershipUser,
				false,
				false,
			))
		}
	}
	impacts = append(impacts, pathImpact(
		"ReShade.log",
		PathRoleLog,
		"ignore",
		OwnershipUser,
		pathExists(filepath.Join(targetPath, "ReShade.log")),
		true,
	))
	userContent, warnings, err := inventoryUserContent(gameRoot, targetPath)
	if err != nil {
		return Preview{}, err
	}
	userContent = mergePreparedUserContent(gameRoot, targetPath, userContent, setupResult.Prepared.Files, update)
	impacts = append(impacts, userContentImpacts(userContent)...)
	preview := Preview{
		Operations:  operations,
		PathImpacts: impacts,
		Warnings:    warnings,
		Conflicts:   []string{},
		Drift:       []Drift{},
		DesiredTarget: &TargetState{
			RuntimeVersion:   artifact.Version,
			Provenance:       provenanceFromArtifact(artifact),
			ManagementOrigin: "installed",
			Manifest: Manifest{
				Version:                    ManifestVersion,
				Files:                      manifestFiles,
				HasPreAdoptionRollbackData: false,
				VariantProvenance:          VariantProvenanceVerified,
				UserContent:                userContent,
			},
		},
	}
	if update && existing != nil {
		oldManifest, decodeErr := DecodeManifest(existing.ManifestJSON)
		if decodeErr != nil {
			return Preview{}, decodeErr
		}
		preview.DesiredTarget.Manifest.Files = retainUserOwnedManifestFiles(
			oldManifest.Files,
			preview.DesiredTarget.Manifest.Files,
		)
		if existing.RenderingAPI != string(request.RenderingAPI) ||
			!strings.EqualFold(existing.ProxyFilename, request.ProxyFilename) {
			preview.Warnings = append(preview.Warnings,
				"The update changes the managed DirectX proxy layout.")
		}
		if existing.BuildVariant != string(request.BuildVariant) {
			preview.Warnings = append(preview.Warnings,
				"The update changes the ReShade build variant.")
		}
	}
	return preview, nil
}

func (p *directXPlanner) planAdopt(
	gameRoot string,
	targetPath string,
	request Request,
) (Preview, error) {
	runtimePath := filepath.Join(targetPath, request.ProxyFilename)
	metadata, err := p.readMetadata(runtimePath)
	if err != nil {
		return Preview{}, err
	}
	if !isReShadeMetadata(metadata) {
		return Preview{
			Operations:       []Operation{},
			PathImpacts:      []PathImpact{},
			Warnings:         []string{},
			Conflicts:        []string{"The selected proxy is not positively identified as ReShade."},
			Drift:            []Drift{},
			UserContentDrift: []UserContentDrift{},
		}, nil
	}
	architecture, err := p.inspectArchitecture(runtimePath)
	if err != nil {
		return Preview{}, err
	}
	if architecture != request.Architecture {
		return Preview{}, errors.New("existing ReShade runtime architecture does not match the selected executable")
	}
	version := runtimeVersionFromMetadata(metadata)
	if version == "" {
		return Preview{}, errors.New("existing ReShade runtime version could not be identified")
	}
	hash, size, err := fileops.FileIntegrity(runtimePath)
	if err != nil {
		return Preview{}, err
	}
	userContent, warnings, err := inventoryUserContent(gameRoot, targetPath)
	if err != nil {
		return Preview{}, err
	}
	warnings = append(warnings,
		"The build variant is user-declared and cannot be cryptographically verified.",
		"Exact pre-adoption rollback is unavailable; uninstall removes only the adopted runtime.",
	)
	return Preview{
		Operations: []Operation{
			{
				Type:       "adopt",
				TargetPath: runtimePath,
				SHA256:     hash,
				SizeBytes:  size,
			},
		},
		PathImpacts: append(
			[]PathImpact{
				pathImpact(
					request.ProxyFilename,
					PathRoleRuntime,
					"adopt",
					OwnershipAdopted,
					true,
					false,
				),
			},
			userContentImpacts(userContent)...,
		),
		Warnings:         warnings,
		Conflicts:        []string{},
		Drift:            []Drift{},
		UserContentDrift: []UserContentDrift{},
		DesiredTarget: &TargetState{
			RuntimeVersion:   version,
			ManagementOrigin: "adopted",
			Manifest: Manifest{
				Version: ManifestVersion,
				Files: []ManagedFile{
					{
						RelativePath: request.ProxyFilename,
						SHA256:       hash,
						SizeBytes:    size,
						Ownership:    OwnershipAdopted,
					},
				},
				HasPreAdoptionRollbackData: false,
				VariantProvenance:          VariantProvenanceUserDeclared,
				UserContent:                userContent,
			},
		},
	}, nil
}

func (p *directXPlanner) planRepair(
	ctx context.Context,
	gameRoot string,
	targetPath string,
	executablePath string,
	request Request,
	existing *dbtypes.ReShadeTarget,
) (Preview, error) {
	if existing == nil {
		return Preview{}, errors.New("managed target is required for repair")
	}
	if err := requireExistingLayout(request, existing); err != nil {
		return Preview{}, err
	}
	manifest, err := DecodeManifest(existing.ManifestJSON)
	if err != nil {
		return Preview{}, err
	}
	runtime, found := runtimeManifestFile(manifest, existing.ProxyFilename)
	if !found {
		return Preview{}, errors.New("managed runtime is missing from the ownership manifest")
	}
	runtimePath := filepath.Join(targetPath, existing.ProxyFilename)
	matches, matchErr := fileops.FileMatchesIntegrity(runtimePath, runtime.SHA256, runtime.SizeBytes)
	if matchErr == nil && matches {
		userContent, warnings, inventoryErr := inventoryUserContent(gameRoot, targetPath)
		if inventoryErr != nil {
			return Preview{}, inventoryErr
		}
		manifest.Version = ManifestVersion
		manifest.UserContent = userContent
		return Preview{
			Operations:       []Operation{},
			PathImpacts:      append([]PathImpact{pathImpact(existing.ProxyFilename, PathRoleRuntime, "verify", runtime.Ownership, true, false)}, userContentImpacts(userContent)...),
			Warnings:         warnings,
			Conflicts:        []string{},
			Drift:            []Drift{},
			UserContentDrift: []UserContentDrift{},
			DesiredTarget: &TargetState{
				RuntimeVersion:   existing.RuntimeVersion,
				Provenance:       provenanceFromRow(existing),
				ManagementOrigin: existing.ManagementOrigin,
				Manifest:         manifest,
			},
		}, nil
	}
	release, err := releaseFromRow(existing)
	if err != nil {
		return Preview{}, err
	}
	artifact, err := p.acquireInstaller(ctx, release, InstallerAcquireOptions{
		Acknowledgements: installerAcknowledgements(request),
	})
	if err != nil {
		return Preview{}, err
	}
	setupResult, err := p.prepareSetup(ctx, SetupRequest{
		Artifact:                    artifact,
		TargetExecutable:            executablePath,
		RenderingAPI:                request.RenderingAPI,
		Operation:                   SetupOperationInstall,
		Architecture:                request.Architecture,
		ExpectedProxy:               request.ProxyFilename,
		ExpectedOutputRelativePaths: []string{request.ProxyFilename, "ReShade.ini", "ReShade.log", "ReShadePreset.ini"},
		Acknowledgements:            installerAcknowledgements(request),
	}, SetupRunnerOptions{})
	if err != nil {
		return Preview{}, err
	}
	if setupResult.Prepared == nil {
		return Preview{}, errors.New("official ReShade setup produced no repair output")
	}
	prepared, ok := preparedFilesByRelativePath(setupResult.Prepared.Files)[strings.ToLower(request.ProxyFilename)]
	if !ok {
		return Preview{}, errors.New("prepared ReShade repair runtime is missing")
	}
	userContent, warnings, err := inventoryUserContent(gameRoot, targetPath)
	if err != nil {
		return Preview{}, err
	}
	runtime.SHA256 = prepared.SHA256
	runtime.SizeBytes = prepared.SizeBytes
	replaceManifestFile(&manifest, runtime)
	manifest.Version = ManifestVersion
	manifest.UserContent = userContent
	return Preview{
		Operations: []Operation{
			{
				Type:       "copy",
				SourcePath: prepared.Path,
				TargetPath: runtimePath,
				SHA256:     prepared.SHA256,
				SizeBytes:  prepared.SizeBytes,
			},
		},
		PathImpacts: append(
			[]PathImpact{pathImpact(existing.ProxyFilename, PathRoleRuntime, "repair", runtime.Ownership, pathExists(runtimePath), false)},
			userContentImpacts(userContent)...,
		),
		Warnings:         warnings,
		Conflicts:        []string{},
		Drift:            []Drift{},
		UserContentDrift: []UserContentDrift{},
		DesiredTarget: &TargetState{
			RuntimeVersion:   existing.RuntimeVersion,
			Provenance:       provenanceFromRow(existing),
			ManagementOrigin: existing.ManagementOrigin,
			Manifest:         manifest,
		},
	}, nil
}

func (p *directXPlanner) planUninstall(
	gameRoot string,
	targetPath string,
	request Request,
	existing *dbtypes.ReShadeTarget,
) (Preview, error) {
	_ = gameRoot
	if existing == nil {
		return Preview{}, errors.New("managed target is required for uninstall")
	}
	if err := requireExistingLayout(request, existing); err != nil {
		return Preview{}, err
	}
	manifest, err := DecodeManifest(existing.ManifestJSON)
	if err != nil {
		return Preview{}, err
	}
	runtime, found := runtimeManifestFile(manifest, existing.ProxyFilename)
	if !found {
		return Preview{}, errors.New("managed runtime is missing from the ownership manifest")
	}
	target := filepath.Join(targetPath, runtime.RelativePath)
	operation := Operation{
		Type:       "delete",
		TargetPath: target,
	}
	action := "remove"
	if runtime.BackupPath != nil && runtime.BackupSHA256 != nil && runtime.BackupSize != nil {
		matches, matchErr := fileops.FileMatchesIntegrity(
			*runtime.BackupPath,
			*runtime.BackupSHA256,
			*runtime.BackupSize,
		)
		if matchErr == nil && matches {
			operation = Operation{
				Type:       "restore",
				SourcePath: *runtime.BackupPath,
				TargetPath: target,
				SHA256:     *runtime.BackupSHA256,
				SizeBytes:  *runtime.BackupSize,
			}
			action = "restore backup"
		}
	}
	impacts := []PathImpact{
		pathImpact(runtime.RelativePath, PathRoleRuntime, action, runtime.Ownership, pathExists(target), false),
	}
	operations := []Operation{operation}
	for _, file := range manifest.Files {
		if strings.EqualFold(filepath.Clean(file.RelativePath), filepath.Clean(runtime.RelativePath)) {
			continue
		}
		if file.Ownership != OwnershipManaged && file.Ownership != OwnershipAdopted {
			continue
		}
		if file.Role == PathRoleRuntime {
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
			file.Ownership,
			pathExists(target),
			false,
		))
	}
	for _, content := range manifest.UserContent {
		impacts = append(impacts, pathImpact(
			content.Path,
			content.Role,
			"preserve",
			OwnershipUser,
			content.Exists,
			true,
		))
	}
	return Preview{
		Operations:       operations,
		PathImpacts:      impacts,
		Warnings:         []string{},
		Conflicts:        []string{},
		Drift:            []Drift{},
		UserContentDrift: []UserContentDrift{},
	}, nil
}

func (p *directXPlanner) proxyConflicts(
	targetPath string,
	request Request,
	existing *dbtypes.ReShadeTarget,
) ([]string, []PathImpact) {
	var conflicts []string
	var impacts []PathImpact
	reShadeCount := 0
	for _, filename := range supportedDirectXProxies {
		path := filepath.Join(targetPath, filename)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() {
			conflicts = append(conflicts, fmt.Sprintf("DirectX proxy %q cannot be safely inspected.", filename))
			impacts = append(impacts, blockingProxyImpact(filename))
			continue
		}
		metadata, metadataErr := p.readMetadata(path)
		isReShade := metadataErr == nil && isReShadeMetadata(metadata)
		if isReShade {
			reShadeCount++
		}
		allowedExisting := existing != nil && strings.EqualFold(existing.ProxyFilename, filename)
		switch request.Action {
		case ActionAdopt:
			if !strings.EqualFold(request.ProxyFilename, filename) || !isReShade {
				conflicts = append(conflicts, fmt.Sprintf("Existing DirectX proxy %q blocks adoption.", filename))
				impacts = append(impacts, blockingProxyImpact(filename))
			}
		case ActionInstall:
			if isReShade {
				conflicts = append(conflicts,
					fmt.Sprintf("Existing unmanaged ReShade proxy %q must be adopted instead of overwritten.", filename))
			} else {
				conflicts = append(conflicts, fmt.Sprintf("Existing foreign DirectX proxy %q blocks install.", filename))
			}
			impacts = append(impacts, blockingProxyImpact(filename))
		default:
			if !allowedExisting {
				conflicts = append(conflicts, fmt.Sprintf("Additional DirectX proxy %q blocks managed mutation.", filename))
				impacts = append(impacts, blockingProxyImpact(filename))
			}
		}
	}
	if reShadeCount > 1 {
		conflicts = append(conflicts, "Multiple ReShade DirectX proxies were detected in the target.")
	}
	return conflicts, impacts
}

func blockingProxyImpact(filename string) PathImpact {
	return PathImpact{
		Path:      filename,
		Role:      PathRoleRuntime,
		Action:    "block",
		Ownership: OwnershipForeign,
		Exists:    true,
		Blocking:  true,
	}
}

func installerVariant(variant BuildVariant) InstallerVariant {
	if variant == BuildVariantAddon {
		return InstallerVariantAddon
	}
	return InstallerVariantStandard
}

func installerAcknowledgements(request Request) InstallerAcknowledgements {
	return InstallerAcknowledgements{
		SinglePlayerAcknowledged:  request.SinglePlayerAcknowledged,
		AntiCheatRiskAcknowledged: request.AntiCheatRiskAcknowledged,
	}
}

func setupExistingInputs(targetPath string, proxyFilename string) ([]SetupInput, error) {
	var inputs []SetupInput
	for _, relativePath := range []string{proxyFilename, "ReShade.ini", "ReShadePreset.ini"} {
		sourcePath := filepath.Join(targetPath, relativePath)
		info, err := os.Stat(sourcePath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		inputs = append(inputs, SetupInput{
			SourcePath:   sourcePath,
			RelativePath: relativePath,
		})
	}
	return inputs, nil
}

func preparedFilesByRelativePath(files []PreparedSetupFile) map[string]PreparedSetupFile {
	result := make(map[string]PreparedSetupFile, len(files))
	for _, file := range files {
		result[strings.ToLower(filepath.Clean(file.RelativePath))] = file
	}
	return result
}

func managedFileFromPrepared(file PreparedSetupFile, ownership Ownership) ManagedFile {
	return ManagedFile{
		RelativePath: file.RelativePath,
		SHA256:       file.SHA256,
		SizeBytes:    file.SizeBytes,
		Ownership:    ownership,
	}
}

func provenanceFromArtifact(artifact InstallerArtifact) InstallerProvenance {
	tag := "v" + artifact.Version
	assetName := artifact.AssetName
	url := artifact.URL
	digest := artifact.SHA256
	size := artifact.SizeBytes
	return InstallerProvenance{
		Tag:       &tag,
		AssetName: &assetName,
		URL:       &url,
		Digest:    &digest,
		Size:      &size,
	}
}

func provenanceFromRow(row *dbtypes.ReShadeTarget) InstallerProvenance {
	return InstallerProvenance{
		Tag:       row.InstallerTag,
		AssetName: row.InstallerAssetName,
		URL:       row.InstallerURL,
		Digest:    row.InstallerDigest,
		Size:      row.InstallerSize,
	}
}

func releaseFromRow(row *dbtypes.ReShadeTarget) (InstallerRelease, error) {
	version := strings.TrimSpace(row.RuntimeVersion)
	if version == "" {
		return InstallerRelease{}, errors.New("recorded ReShade runtime version is missing")
	}
	variant := installerVariant(BuildVariant(row.BuildVariant))
	assetName := installerFileName(version, variant)
	url, err := installerDownloadURL(DefaultDownloadBaseURL, version, variant)
	if err != nil {
		return InstallerRelease{}, err
	}
	if row.InstallerAssetName != nil && strings.TrimSpace(*row.InstallerAssetName) != "" {
		assetName = strings.TrimSpace(*row.InstallerAssetName)
	}
	if row.InstallerURL != nil && strings.TrimSpace(*row.InstallerURL) != "" {
		url = strings.TrimSpace(*row.InstallerURL)
	}
	return InstallerRelease{
		Version:   version,
		Variant:   variant,
		AssetName: assetName,
		URL:       url,
	}, nil
}

func mergePreparedUserContent(
	gameRoot string,
	targetPath string,
	current []UserContent,
	prepared []PreparedSetupFile,
	update bool,
) []UserContent {
	if update {
		return current
	}
	for _, file := range prepared {
		role := PathRole("")
		switch strings.ToLower(filepath.Clean(file.RelativePath)) {
		case "reshade.ini":
			role = PathRoleConfiguration
		case "reshadepreset.ini":
			role = PathRolePreset
		default:
			continue
		}
		target := filepath.Join(targetPath, file.RelativePath)
		if pathExists(target) {
			continue
		}
		relative, err := filepath.Rel(gameRoot, target)
		if err != nil {
			continue
		}
		current = append(current, UserContent{
			Path:      relative,
			Role:      role,
			SHA256:    file.SHA256,
			SizeBytes: file.SizeBytes,
			Exists:    true,
		})
	}
	return deduplicateUserContent(current)
}

func pathImpact(
	path string,
	role PathRole,
	action string,
	ownership Ownership,
	exists bool,
	preservationOnly bool,
) PathImpact {
	return PathImpact{
		Path:             path,
		Role:             role,
		Action:           action,
		Ownership:        ownership,
		Exists:           exists,
		PreservationOnly: preservationOnly,
	}
}

func userContentImpacts(content []UserContent) []PathImpact {
	result := make([]PathImpact, 0, len(content))
	for _, item := range content {
		result = append(result, pathImpact(
			item.Path,
			item.Role,
			"preserve",
			OwnershipUser,
			item.Exists,
			true,
		))
	}
	return result
}

func retainUserOwnedManifestFiles(existing []ManagedFile, desired []ManagedFile) []ManagedFile {
	result := append([]ManagedFile(nil), desired...)
	for _, file := range existing {
		if file.Ownership == OwnershipUser {
			result = append(result, file)
		}
	}
	return result
}

func requireExistingLayout(request Request, existing *dbtypes.ReShadeTarget) error {
	if existing.ExecutableRelativePath != request.ExecutableRelativePath ||
		existing.RenderingAPI != string(request.RenderingAPI) ||
		!strings.EqualFold(existing.ProxyFilename, request.ProxyFilename) ||
		existing.Architecture != string(request.Architecture) ||
		existing.BuildVariant != string(request.BuildVariant) {
		return errors.New("repair and uninstall requests must match the persisted target layout")
	}
	return nil
}

func runtimeManifestFile(manifest Manifest, proxyFilename string) (ManagedFile, bool) {
	for _, file := range manifest.Files {
		if strings.EqualFold(filepath.Clean(file.RelativePath), filepath.Clean(proxyFilename)) &&
			(file.Ownership == OwnershipManaged || file.Ownership == OwnershipAdopted) {
			return file, true
		}
	}
	return ManagedFile{}, false
}

func replaceManifestFile(manifest *Manifest, replacement ManagedFile) {
	for index := range manifest.Files {
		if strings.EqualFold(
			filepath.Clean(manifest.Files[index].RelativePath),
			filepath.Clean(replacement.RelativePath),
		) {
			manifest.Files[index] = replacement
			return
		}
	}
	manifest.Files = append(manifest.Files, replacement)
}
