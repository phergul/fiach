package reshade

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/phergul/fiach/internal/fileops"
	"github.com/phergul/fiach/internal/winversion"
)

type setupProcessRunnerFunc func(context.Context, setupProcessRequest) (setupProcessResult, error)

func (function setupProcessRunnerFunc) RunSetupProcess(
	ctx context.Context,
	request setupProcessRequest,
) (setupProcessResult, error) {
	return function(ctx, request)
}

func TestPrepareSetupStagesVerifiesAndReusesCache(t *testing.T) {
	t.Parallel()

	request, verifier := setupTestRequest(t)
	var calls int
	var gotArguments []string
	processRunner := setupProcessRunnerFunc(func(
		_ context.Context,
		processRequest setupProcessRequest,
	) (setupProcessResult, error) {
		calls++
		gotArguments = append([]string(nil), processRequest.Arguments...)
		proxyPath := filepath.Join(processRequest.WorkingDir, request.ExpectedProxy)
		if err := os.WriteFile(proxyPath, []byte("runtime"), 0o644); err != nil {
			return setupProcessResult{}, err
		}
		now := time.Now()
		return setupProcessResult{
			ExitCode:   0,
			StartedAt:  now,
			FinishedAt: now.Add(time.Second),
			Stdout:     "installed",
		}, nil
	})
	options := setupTestRunnerOptions(t, verifier, processRunner)

	result, err := PrepareSetup(context.Background(), request, options)
	if err != nil {
		t.Fatalf("PrepareSetup() error = %v", err)
	}
	if result.Prepared == nil || len(result.Prepared.Files) != 1 {
		t.Fatalf("Prepared = %+v", result.Prepared)
	}
	wantArguments := []string{
		filepath.Join(result.Prepared.WorkspacePath, filepath.Base(request.TargetExecutable)),
		"--headless",
		"--api",
		"d3d11",
	}
	if !reflect.DeepEqual(gotArguments, wantArguments) {
		t.Fatalf("arguments = %#v, want %#v", gotArguments, wantArguments)
	}
	if result.Execution.Stdout != "installed" || result.Execution.Cached {
		t.Fatalf("Execution = %+v", result.Execution)
	}

	cached, err := PrepareSetup(context.Background(), request, options)
	if err != nil {
		t.Fatalf("cached PrepareSetup() error = %v", err)
	}
	if calls != 1 || cached.Prepared == nil || !cached.Execution.Cached {
		t.Fatalf("calls = %d, cached = %+v", calls, cached)
	}
}

func TestPrepareSetupUpdateArgumentsPreserveHostilePath(t *testing.T) {
	t.Parallel()

	request, verifier := setupTestRequest(t)
	request.Operation = SetupOperationUpdate
	request.TargetExecutable = filepath.Join(t.TempDir(), "Game & (Tools).exe")
	if err := os.WriteFile(request.TargetExecutable, []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	processRunner := setupProcessRunnerFunc(func(
		_ context.Context,
		processRequest setupProcessRequest,
	) (setupProcessResult, error) {
		wantSuffix := []string{"--headless", "--api", "d3d11", "--state", "update"}
		if !reflect.DeepEqual(processRequest.Arguments[1:], wantSuffix) {
			t.Fatalf("arguments = %#v", processRequest.Arguments)
		}
		if filepath.Base(processRequest.Arguments[0]) != filepath.Base(request.TargetExecutable) {
			t.Fatalf("target argument = %q", processRequest.Arguments[0])
		}
		if err := os.WriteFile(
			filepath.Join(processRequest.WorkingDir, request.ExpectedProxy),
			[]byte("runtime"),
			0o644,
		); err != nil {
			return setupProcessResult{}, err
		}
		now := time.Now()
		return setupProcessResult{StartedAt: now, FinishedAt: now, ExitCode: 0}, nil
	})

	if _, err := PrepareSetup(
		context.Background(),
		request,
		setupTestRunnerOptions(t, verifier, processRunner),
	); err != nil {
		t.Fatalf("PrepareSetup() error = %v", err)
	}
}

func TestPrepareSetupRejectsUnexpectedWorkspaceMutation(t *testing.T) {
	t.Parallel()

	request, verifier := setupTestRequest(t)
	processRunner := setupProcessRunnerFunc(func(
		_ context.Context,
		processRequest setupProcessRequest,
	) (setupProcessResult, error) {
		_ = os.WriteFile(
			filepath.Join(processRequest.WorkingDir, request.ExpectedProxy),
			[]byte("runtime"),
			0o644,
		)
		_ = os.WriteFile(
			filepath.Join(processRequest.WorkingDir, "unexpected.dll"),
			[]byte("unexpected"),
			0o644,
		)
		now := time.Now()
		return setupProcessResult{StartedAt: now, FinishedAt: now, ExitCode: 0}, nil
	})
	options := setupTestRunnerOptions(t, verifier, processRunner)

	result, err := PrepareSetup(context.Background(), request, options)
	if err == nil || !strings.Contains(err.Error(), "changed undeclared paths") {
		t.Fatalf("PrepareSetup() error = %v", err)
	}
	if result.Prepared != nil {
		t.Fatalf("Prepared = %+v, want nil", result.Prepared)
	}
	entries, readErr := os.ReadDir(options.WorkspaceRoot)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatalf("workspace entries = %d, want cleanup", len(entries))
	}
}

func TestPrepareSetupReturnsStructuredProcessFailure(t *testing.T) {
	t.Parallel()

	request, verifier := setupTestRequest(t)
	now := time.Now()
	processRunner := setupProcessRunnerFunc(func(
		context.Context,
		setupProcessRequest,
	) (setupProcessResult, error) {
		return setupProcessResult{
			ExitCode:        -1,
			StartedAt:       now,
			FinishedAt:      now.Add(time.Second),
			Stdout:          "partial",
			Stderr:          "cancelled",
			StdoutTruncated: true,
			Cancelled:       true,
		}, context.Canceled
	})

	result, err := PrepareSetup(
		context.Background(),
		request,
		setupTestRunnerOptions(t, verifier, processRunner),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PrepareSetup() error = %v, want cancellation", err)
	}
	if !result.Execution.Cancelled ||
		!result.Execution.StdoutTruncated ||
		result.Execution.Stdout != "partial" ||
		result.Execution.Stderr != "cancelled" {
		t.Fatalf("Execution = %+v", result.Execution)
	}
}

func TestPrepareSetupRejectsRuntimeVerificationMismatch(t *testing.T) {
	t.Parallel()

	request, verifier := setupTestRequest(t)
	processRunner := setupProcessRunnerFunc(func(
		_ context.Context,
		processRequest setupProcessRequest,
	) (setupProcessResult, error) {
		if err := os.WriteFile(
			filepath.Join(processRequest.WorkingDir, request.ExpectedProxy),
			[]byte("runtime"),
			0o644,
		); err != nil {
			return setupProcessResult{}, err
		}
		now := time.Now()
		return setupProcessResult{StartedAt: now, FinishedAt: now, ExitCode: 0}, nil
	})
	options := setupTestRunnerOptions(t, verifier, processRunner)
	options.ReadMetadata = func(string) (winversion.Metadata, error) {
		return winversion.Metadata{
			ProductName:      "Not ReShade",
			OriginalFilename: "ReShade64.dll",
			ProductVersion:   "6.7.3",
		}, nil
	}

	_, err := PrepareSetup(context.Background(), request, options)
	if err == nil || !strings.Contains(err.Error(), "is not ReShade") {
		t.Fatalf("PrepareSetup() error = %v", err)
	}
}

func setupTestRequest(
	t *testing.T,
) (SetupRequest, InstallerSignatureVerifier) {
	t.Helper()
	root := t.TempDir()
	installerPath := filepath.Join(root, "ReShade_Setup_6.7.3.exe")
	targetPath := filepath.Join(root, "Game Folder", "Game.exe")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(installerPath, []byte("installer"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("game"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash, size, err := fileops.FileIntegrity(installerPath)
	if err != nil {
		t.Fatal(err)
	}
	signature := InstallerSignature{
		Status:     InstallerSignatureStatusVerified,
		Subject:    reShadeSignerSubject,
		SPKISHA256: reShadeSignerSPKISHA256,
	}
	verifier := signatureVerifierFunc(func(string, InstallerVariant) (InstallerSignature, error) {
		return signature, nil
	})
	return SetupRequest{
		Artifact: InstallerArtifact{
			InstallerRelease: InstallerRelease{
				Version:   "6.7.3",
				Variant:   InstallerVariantStandard,
				AssetName: "ReShade_Setup_6.7.3.exe",
				URL:       "https://reshade.me/downloads/ReShade_Setup_6.7.3.exe",
			},
			Path:      installerPath,
			SizeBytes: size,
			SHA256:    hash,
			Signature: signature,
		},
		TargetExecutable:            targetPath,
		RenderingAPI:                RenderingAPID3D11,
		Operation:                   SetupOperationInstall,
		Architecture:                ArchitectureX64,
		ExpectedProxy:               "dxgi.dll",
		ExpectedOutputRelativePaths: []string{"dxgi.dll"},
	}, verifier
}

func setupTestRunnerOptions(
	t *testing.T,
	verifier InstallerSignatureVerifier,
	processRunner setupProcessRunner,
) SetupRunnerOptions {
	t.Helper()
	return SetupRunnerOptions{
		WorkspaceRoot:     t.TempDir(),
		ProcessRunner:     processRunner,
		SignatureVerifier: verifier,
		ReadMetadata: func(string) (winversion.Metadata, error) {
			return winversion.Metadata{
				ProductName:      "ReShade",
				OriginalFilename: "ReShade64.dll",
				ProductVersion:   "6.7.3.123",
			}, nil
		},
		InspectArchitecture: func(string) (Architecture, error) {
			return ArchitectureX64, nil
		},
	}
}
