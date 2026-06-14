package reshade

import (
	"context"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/winversion"
)

const (
	DefaultSetupTimeout          = 5 * time.Minute
	DefaultSetupOutputLimitBytes = 1024 * 1024
	preparedSetupManifestVersion = 1
)

var setupWorkspaceMu sync.Mutex

type SetupOperation string

const (
	SetupOperationInstall SetupOperation = "install"
	SetupOperationUpdate  SetupOperation = "update"
)

type SetupInput struct {
	SourcePath   string `json:"sourcePath"`
	RelativePath string `json:"relativePath"`
}

type SetupRequest struct {
	Artifact                    InstallerArtifact
	TargetExecutable            string
	RenderingAPI                RenderingAPI
	Operation                   SetupOperation
	Architecture                Architecture
	ExpectedProxy               string
	ExistingInputs              []SetupInput
	ExpectedOutputRelativePaths []string
	Acknowledgements            InstallerAcknowledgements
	Timeout                     time.Duration
}

type SetupExecutionResult struct {
	Arguments       []string      `json:"arguments"`
	ExitCode        int           `json:"exitCode"`
	StartedAt       time.Time     `json:"startedAt"`
	FinishedAt      time.Time     `json:"finishedAt"`
	Duration        time.Duration `json:"duration"`
	Stdout          string        `json:"stdout"`
	Stderr          string        `json:"stderr"`
	StdoutTruncated bool          `json:"stdoutTruncated"`
	StderrTruncated bool          `json:"stderrTruncated"`
	Cancelled       bool          `json:"cancelled"`
	TimedOut        bool          `json:"timedOut"`
	ElevationNeeded bool          `json:"elevationNeeded"`
	Cached          bool          `json:"cached"`
}

type PreparedSetupFile struct {
	RelativePath string `json:"relativePath"`
	Path         string `json:"path"`
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"sizeBytes"`
}

type PreparedSetup struct {
	WorkspacePath string               `json:"workspacePath"`
	Artifact      InstallerArtifact    `json:"artifact"`
	ExecutableSHA string               `json:"executableSha256"`
	RenderingAPI  RenderingAPI         `json:"renderingApi"`
	Operation     SetupOperation       `json:"operation"`
	Architecture  Architecture         `json:"architecture"`
	ExpectedProxy string               `json:"expectedProxy"`
	Files         []PreparedSetupFile  `json:"files"`
	Execution     SetupExecutionResult `json:"execution"`
}

type SetupRunResult struct {
	Execution SetupExecutionResult `json:"execution"`
	Prepared  *PreparedSetup       `json:"prepared,omitempty"`
}

type setupProcessRequest struct {
	ExecutablePath string
	Arguments      []string
	WorkingDir     string
	Timeout        time.Duration
	OutputLimit    int
}

type setupProcessResult struct {
	ExitCode        int
	StartedAt       time.Time
	FinishedAt      time.Time
	Stdout          string
	Stderr          string
	StdoutTruncated bool
	StderrTruncated bool
	Cancelled       bool
	TimedOut        bool
	ElevationNeeded bool
}

type setupProcessRunner interface {
	RunSetupProcess(context.Context, setupProcessRequest) (setupProcessResult, error)
}

type SetupRunnerOptions struct {
	WorkspaceRoot       string
	ProcessRunner       setupProcessRunner
	SignatureVerifier   InstallerSignatureVerifier
	ReadMetadata        func(string) (winversion.Metadata, error)
	InspectArchitecture func(string) (Architecture, error)
	OutputLimitBytes    int
}

type preparedSetupManifest struct {
	Version  int           `json:"version"`
	Prepared PreparedSetup `json:"prepared"`
}

type workspaceFile struct {
	SHA256 string
	Size   int64
}

func PrepareSetup(
	ctx context.Context,
	request SetupRequest,
	options SetupRunnerOptions,
) (result SetupRunResult, err error) {
	setupWorkspaceMu.Lock()
	defer setupWorkspaceMu.Unlock()

	defer func() {
		if err != nil {
			err = fmt.Errorf("prepare managed ReShade setup: %w", err)
		}
	}()
	options = normalizeSetupRunnerOptions(options)
	cleanRequest, executableSHA, err := validateSetupRequest(request, options.SignatureVerifier)
	if err != nil {
		return SetupRunResult{}, err
	}
	workspaceKey, err := setupWorkspaceKey(cleanRequest, executableSHA)
	if err != nil {
		return SetupRunResult{}, err
	}
	workspacePath := filepath.Join(options.WorkspaceRoot, workspaceKey)
	manifestPath := filepath.Join(workspacePath, "prepared.json")
	if prepared, cacheErr := readPreparedSetup(manifestPath, cleanRequest, executableSHA); cacheErr == nil {
		prepared.Execution.Cached = true
		return SetupRunResult{
			Execution: prepared.Execution,
			Prepared:  &prepared,
		}, nil
	}
	if err := os.RemoveAll(workspacePath); err != nil {
		return SetupRunResult{}, fmt.Errorf("reset ReShade setup workspace: %w", err)
	}
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		return SetupRunResult{}, fmt.Errorf("create ReShade setup workspace: %w", err)
	}
	success := false
	defer func() {
		if !success {
			_ = os.RemoveAll(workspacePath)
		}
	}()

	stagedExecutable := filepath.Join(workspacePath, filepath.Base(cleanRequest.TargetExecutable))
	if err := copySetupFile(cleanRequest.TargetExecutable, stagedExecutable); err != nil {
		return SetupRunResult{}, err
	}
	for _, input := range cleanRequest.ExistingInputs {
		target := filepath.Join(workspacePath, input.RelativePath)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return SetupRunResult{}, fmt.Errorf("create staged input directory: %w", err)
		}
		if err := copySetupFile(input.SourcePath, target); err != nil {
			return SetupRunResult{}, err
		}
	}
	before, err := inventoryWorkspace(workspacePath)
	if err != nil {
		return SetupRunResult{}, err
	}

	arguments := setupArguments(stagedExecutable, cleanRequest.RenderingAPI, cleanRequest.Operation)
	processResult, processErr := options.ProcessRunner.RunSetupProcess(ctx, setupProcessRequest{
		ExecutablePath: cleanRequest.Artifact.Path,
		Arguments:      arguments,
		WorkingDir:     workspacePath,
		Timeout:        cleanRequest.Timeout,
		OutputLimit:    options.OutputLimitBytes,
	})
	execution := setupExecutionResult(arguments, processResult)
	result.Execution = execution
	if processErr != nil {
		return result, processErr
	}
	if processResult.ExitCode != 0 {
		return result, fmt.Errorf("ReShade setup exited with code %d", processResult.ExitCode)
	}
	if processResult.ElevationNeeded {
		return result, errors.New("ReShade setup requires elevation")
	}

	after, err := inventoryWorkspace(workspacePath)
	if err != nil {
		return result, err
	}
	if err := validateWorkspaceChanges(
		before,
		after,
		cleanRequest.ExpectedOutputRelativePaths,
	); err != nil {
		return result, err
	}
	files, err := verifyPreparedOutputs(workspacePath, cleanRequest, options)
	if err != nil {
		return result, err
	}
	prepared := PreparedSetup{
		WorkspacePath: workspacePath,
		Artifact:      cleanRequest.Artifact,
		ExecutableSHA: executableSHA,
		RenderingAPI:  cleanRequest.RenderingAPI,
		Operation:     cleanRequest.Operation,
		Architecture:  cleanRequest.Architecture,
		ExpectedProxy: cleanRequest.ExpectedProxy,
		Files:         files,
		Execution:     execution,
	}
	if err := writePreparedSetup(manifestPath, prepared); err != nil {
		return result, err
	}
	success = true
	result.Prepared = &prepared
	return result, nil
}

func normalizeSetupRunnerOptions(options SetupRunnerOptions) SetupRunnerOptions {
	if options.WorkspaceRoot == "" {
		options.WorkspaceRoot = filepath.Join(
			application.Path(application.PathCacheHome),
			"fiach",
			"reshade",
			"prepared",
		)
	}
	if options.ProcessRunner == nil {
		options.ProcessRunner = platformSetupProcessRunner{}
	}
	if options.SignatureVerifier == nil {
		options.SignatureVerifier = platformInstallerSignatureVerifier{}
	}
	if options.ReadMetadata == nil {
		options.ReadMetadata = winversion.Read
	}
	if options.InspectArchitecture == nil {
		options.InspectArchitecture = inspectPEArchitecture
	}
	if options.OutputLimitBytes <= 0 {
		options.OutputLimitBytes = DefaultSetupOutputLimitBytes
	}
	return options
}

func validateSetupRequest(
	request SetupRequest,
	signatureVerifier InstallerSignatureVerifier,
) (SetupRequest, string, error) {
	if !filepath.IsAbs(request.TargetExecutable) {
		return SetupRequest{}, "", errors.New("target executable path must be absolute")
	}
	executableInfo, err := fileops.StatRegularFile("target executable", request.TargetExecutable)
	if err != nil {
		return SetupRequest{}, "", err
	}
	if !strings.EqualFold(filepath.Ext(request.TargetExecutable), ".exe") {
		return SetupRequest{}, "", errors.New("target executable must have an .exe extension")
	}
	_ = executableInfo
	if request.Operation != SetupOperationInstall && request.Operation != SetupOperationUpdate {
		return SetupRequest{}, "", fmt.Errorf("setup operation %q is unsupported", request.Operation)
	}
	if request.RenderingAPI != RenderingAPID3D9 &&
		request.RenderingAPI != RenderingAPID3D10 &&
		request.RenderingAPI != RenderingAPID3D11 &&
		request.RenderingAPI != RenderingAPID3D12 {
		return SetupRequest{}, "", fmt.Errorf("rendering API %q is unsupported", request.RenderingAPI)
	}
	if request.Architecture != ArchitectureX86 && request.Architecture != ArchitectureX64 {
		return SetupRequest{}, "", fmt.Errorf("architecture %q is unsupported", request.Architecture)
	}
	request.ExpectedProxy = strings.TrimSpace(request.ExpectedProxy)
	if !isSupportedDirectXProxy(request.ExpectedProxy) {
		return SetupRequest{}, "", fmt.Errorf("expected proxy %q is unsupported", request.ExpectedProxy)
	}
	if request.Artifact.Path == "" || !filepath.IsAbs(request.Artifact.Path) {
		return SetupRequest{}, "", errors.New("installer artifact path must be absolute")
	}
	if err := validateInstallerRelease(
		request.Artifact.InstallerRelease,
		nil,
		false,
	); err != nil {
		return SetupRequest{}, "", err
	}
	matches, err := fileops.FileMatchesIntegrity(
		request.Artifact.Path,
		request.Artifact.SHA256,
		request.Artifact.SizeBytes,
	)
	if err != nil {
		return SetupRequest{}, "", err
	}
	if !matches {
		return SetupRequest{}, "", errors.New("installer artifact integrity no longer matches")
	}
	signature, err := signatureVerifier.VerifyInstallerSignature(
		request.Artifact.Path,
		request.Artifact.Variant,
	)
	if err != nil {
		return SetupRequest{}, "", err
	}
	if signature != request.Artifact.Signature {
		return SetupRequest{}, "", errors.New("installer artifact signature metadata no longer matches")
	}
	if request.Artifact.Variant == InstallerVariantAddon &&
		(!request.Acknowledgements.SinglePlayerAcknowledged ||
			!request.Acknowledgements.AntiCheatRiskAcknowledged) {
		return SetupRequest{}, "", errors.New(
			"full add-on setup requires separate single-player and anti-cheat risk acknowledgements")
	}
	if request.Timeout <= 0 {
		request.Timeout = DefaultSetupTimeout
	}

	executableSHA, _, err := fileops.FileIntegrity(request.TargetExecutable)
	if err != nil {
		return SetupRequest{}, "", err
	}
	seenPaths := map[string]bool{
		strings.ToLower(filepath.Base(request.TargetExecutable)): true,
	}
	for index := range request.ExistingInputs {
		input := &request.ExistingInputs[index]
		if !filepath.IsAbs(input.SourcePath) {
			return SetupRequest{}, "", fmt.Errorf("existing input %q path must be absolute", input.RelativePath)
		}
		if _, err := fileops.StatRegularFile("existing setup input", input.SourcePath); err != nil {
			return SetupRequest{}, "", err
		}
		cleanRelative, err := cleanSetupRelativePath(input.RelativePath)
		if err != nil {
			return SetupRequest{}, "", err
		}
		key := strings.ToLower(cleanRelative)
		if seenPaths[key] {
			return SetupRequest{}, "", fmt.Errorf("staged path %q is duplicated", cleanRelative)
		}
		seenPaths[key] = true
		input.RelativePath = cleanRelative
	}
	expected := make([]string, 0, len(request.ExpectedOutputRelativePaths)+1)
	expectedSeen := map[string]bool{}
	for _, relativePath := range append(
		append([]string(nil), request.ExpectedOutputRelativePaths...),
		request.ExpectedProxy,
	) {
		cleanRelative, err := cleanSetupRelativePath(relativePath)
		if err != nil {
			return SetupRequest{}, "", err
		}
		key := strings.ToLower(cleanRelative)
		if expectedSeen[key] {
			continue
		}
		expectedSeen[key] = true
		expected = append(expected, cleanRelative)
	}
	sort.Strings(expected)
	request.ExpectedOutputRelativePaths = expected
	return request, executableSHA, nil
}

func cleanSetupRelativePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) {
		return "", fmt.Errorf("setup relative path %q is invalid", path)
	}
	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("setup relative path %q escapes the workspace", path)
	}
	return clean, nil
}

func setupArguments(
	targetExecutable string,
	renderingAPI RenderingAPI,
	operation SetupOperation,
) []string {
	arguments := []string{
		targetExecutable,
		"--headless",
		"--api",
		string(renderingAPI),
	}
	if operation == SetupOperationUpdate {
		arguments = append(arguments, "--state", "update")
	}
	return arguments
}

func setupWorkspaceKey(request SetupRequest, executableSHA string) (string, error) {
	type keyInput struct {
		InstallerSHA   string
		ExecutableSHA  string
		RenderingAPI   RenderingAPI
		Operation      SetupOperation
		Architecture   Architecture
		ExpectedProxy  string
		Inputs         []string
		ExpectedOutput []string
	}
	inputs := make([]string, 0, len(request.ExistingInputs))
	for _, input := range request.ExistingInputs {
		hash, size, err := fileops.FileIntegrity(input.SourcePath)
		if err != nil {
			return "", err
		}
		inputs = append(inputs, fmt.Sprintf("%s:%s:%d", input.RelativePath, hash, size))
	}
	sort.Strings(inputs)
	encoded, err := json.Marshal(keyInput{
		InstallerSHA:   strings.ToLower(request.Artifact.SHA256),
		ExecutableSHA:  strings.ToLower(executableSHA),
		RenderingAPI:   request.RenderingAPI,
		Operation:      request.Operation,
		Architecture:   request.Architecture,
		ExpectedProxy:  strings.ToLower(request.ExpectedProxy),
		Inputs:         inputs,
		ExpectedOutput: request.ExpectedOutputRelativePaths,
	})
	if err != nil {
		return "", fmt.Errorf("encode ReShade setup workspace key: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func copySetupFile(sourcePath string, targetPath string) error {
	return fileops.CopyFileAtomic(fileops.AtomicCopyOptions{
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Mode:       0o644,
		Replace:    false,
		OpenLabel:  "ReShade setup input",
	})
}

func inventoryWorkspace(root string) (map[string]workspaceFile, error) {
	files := map[string]workspaceFile{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("workspace entry %q is not a regular file", path)
		}
		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		hash, size, err := fileops.FileIntegrity(path)
		if err != nil {
			return err
		}
		files[strings.ToLower(filepath.Clean(relativePath))] = workspaceFile{
			SHA256: hash,
			Size:   size,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("inventory ReShade setup workspace: %w", err)
	}
	return files, nil
}

func validateWorkspaceChanges(
	before map[string]workspaceFile,
	after map[string]workspaceFile,
	expectedPaths []string,
) error {
	expected := make(map[string]bool, len(expectedPaths))
	for _, path := range expectedPaths {
		expected[strings.ToLower(filepath.Clean(path))] = true
	}
	changed := map[string]bool{}
	for path, beforeFile := range before {
		afterFile, exists := after[path]
		if !exists || afterFile != beforeFile {
			changed[path] = true
		}
	}
	for path, afterFile := range after {
		beforeFile, exists := before[path]
		if !exists || afterFile != beforeFile {
			changed[path] = true
		}
	}
	var unexpected []string
	for path := range changed {
		if !expected[path] {
			unexpected = append(unexpected, path)
		}
	}
	if len(unexpected) > 0 {
		sort.Strings(unexpected)
		return fmt.Errorf("ReShade setup changed undeclared paths: %s", strings.Join(unexpected, ", "))
	}
	return nil
}

func verifyPreparedOutputs(
	workspacePath string,
	request SetupRequest,
	options SetupRunnerOptions,
) ([]PreparedSetupFile, error) {
	proxyPath := filepath.Join(workspacePath, request.ExpectedProxy)
	if _, err := fileops.StatRegularFile("prepared ReShade proxy", proxyPath); err != nil {
		return nil, err
	}
	metadata, err := options.ReadMetadata(proxyPath)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(strings.TrimSpace(metadata.ProductName), "ReShade") {
		return nil, fmt.Errorf("prepared proxy product name %q is not ReShade", metadata.ProductName)
	}
	expectedOriginalFilename := "ReShade32.dll"
	if request.Architecture == ArchitectureX64 {
		expectedOriginalFilename = "ReShade64.dll"
	}
	if !strings.EqualFold(strings.TrimSpace(metadata.OriginalFilename), expectedOriginalFilename) {
		return nil, fmt.Errorf(
			"prepared proxy original filename %q does not match %q",
			metadata.OriginalFilename,
			expectedOriginalFilename,
		)
	}
	architecture, err := options.InspectArchitecture(proxyPath)
	if err != nil {
		return nil, err
	}
	if architecture != request.Architecture {
		return nil, fmt.Errorf(
			"prepared proxy architecture %q does not match %q",
			architecture,
			request.Architecture,
		)
	}
	runtimeVersion := firstSemanticVersion(metadata.ProductVersion)
	if runtimeVersion == "" {
		runtimeVersion = firstSemanticVersion(metadata.FileVersion)
	}
	if runtimeVersion != request.Artifact.Version {
		return nil, fmt.Errorf(
			"prepared proxy version %q does not match installer version %q",
			runtimeVersion,
			request.Artifact.Version,
		)
	}

	files := make([]PreparedSetupFile, 0, len(request.ExpectedOutputRelativePaths))
	for _, relativePath := range request.ExpectedOutputRelativePaths {
		path := filepath.Join(workspacePath, relativePath)
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat prepared ReShade output %q: %w", relativePath, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("prepared ReShade output %q is not a regular file", relativePath)
		}
		hash, size, err := fileops.FileIntegrity(path)
		if err != nil {
			return nil, err
		}
		files = append(files, PreparedSetupFile{
			RelativePath: relativePath,
			Path:         path,
			SHA256:       hash,
			SizeBytes:    size,
		})
	}
	sort.Slice(files, func(i int, j int) bool {
		return files[i].RelativePath < files[j].RelativePath
	})
	return files, nil
}

func inspectPEArchitecture(path string) (Architecture, error) {
	file, err := pe.Open(path)
	if err != nil {
		return "", fmt.Errorf("inspect PE architecture %q: %w", path, err)
	}
	defer file.Close()
	switch file.FileHeader.Machine {
	case pe.IMAGE_FILE_MACHINE_I386:
		return ArchitectureX86, nil
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return ArchitectureX64, nil
	default:
		return "", fmt.Errorf("PE machine type %#x is unsupported", file.FileHeader.Machine)
	}
}

func firstSemanticVersion(value string) string {
	fields := strings.FieldsFunc(value, func(character rune) bool {
		return (character < '0' || character > '9') && character != '.'
	})
	for _, field := range fields {
		parts := strings.Split(field, ".")
		if len(parts) < 3 {
			continue
		}
		return strings.Join(parts[:3], ".")
	}
	return ""
}

func setupExecutionResult(
	arguments []string,
	result setupProcessResult,
) SetupExecutionResult {
	return SetupExecutionResult{
		Arguments:       append([]string(nil), arguments...),
		ExitCode:        result.ExitCode,
		StartedAt:       result.StartedAt,
		FinishedAt:      result.FinishedAt,
		Duration:        result.FinishedAt.Sub(result.StartedAt),
		Stdout:          result.Stdout,
		Stderr:          result.Stderr,
		StdoutTruncated: result.StdoutTruncated,
		StderrTruncated: result.StderrTruncated,
		Cancelled:       result.Cancelled,
		TimedOut:        result.TimedOut,
		ElevationNeeded: result.ElevationNeeded,
	}
}

func readPreparedSetup(
	manifestPath string,
	request SetupRequest,
	executableSHA string,
) (PreparedSetup, error) {
	contents, err := os.ReadFile(manifestPath)
	if err != nil {
		return PreparedSetup{}, err
	}
	var manifest preparedSetupManifest
	if err := json.Unmarshal(contents, &manifest); err != nil {
		return PreparedSetup{}, fmt.Errorf("decode prepared ReShade setup manifest: %w", err)
	}
	if manifest.Version != preparedSetupManifestVersion {
		return PreparedSetup{}, errors.New("prepared ReShade setup manifest version is unsupported")
	}
	prepared := manifest.Prepared
	if prepared.Artifact != request.Artifact ||
		prepared.ExecutableSHA != executableSHA ||
		prepared.RenderingAPI != request.RenderingAPI ||
		prepared.Operation != request.Operation ||
		prepared.Architecture != request.Architecture ||
		!strings.EqualFold(prepared.ExpectedProxy, request.ExpectedProxy) {
		return PreparedSetup{}, errors.New("prepared ReShade setup cache key does not match")
	}
	for _, file := range prepared.Files {
		if err := fileops.RequirePathWithinRoot(
			"prepared ReShade file",
			file.Path,
			prepared.WorkspacePath,
		); err != nil {
			return PreparedSetup{}, err
		}
		matches, err := fileops.FileMatchesIntegrity(file.Path, file.SHA256, file.SizeBytes)
		if err != nil || !matches {
			return PreparedSetup{}, errors.New("prepared ReShade setup file integrity no longer matches")
		}
	}
	return prepared, nil
}

func writePreparedSetup(path string, prepared PreparedSetup) error {
	contents, err := json.MarshalIndent(preparedSetupManifest{
		Version:  preparedSetupManifestVersion,
		Prepared: prepared,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode prepared ReShade setup manifest: %w", err)
	}
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, contents, 0o644); err != nil {
		return fmt.Errorf("write prepared ReShade setup manifest: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("commit prepared ReShade setup manifest: %w", err)
	}
	return nil
}

func isSupportedDirectXProxy(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "d3d9.dll", "d3d10.dll", "d3d10core.dll", "d3d11.dll", "d3d12.dll", "dxgi.dll":
		return true
	default:
		return false
	}
}
