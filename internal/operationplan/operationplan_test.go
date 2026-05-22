package operationplan

import (
	"reflect"
	"testing"
)

func TestOperationTypeConstants(t *testing.T) {
	t.Parallel()

	if OperationTypeCopy != "copy" {
		t.Fatalf("OperationTypeCopy = %q, want copy", OperationTypeCopy)
	}
	if OperationTypeReplace != "replace" {
		t.Fatalf("OperationTypeReplace = %q, want replace", OperationTypeReplace)
	}
	if OperationTypeCreateDirectory != "create_directory" {
		t.Fatalf("OperationTypeCreateDirectory = %q, want create_directory", OperationTypeCreateDirectory)
	}
}

func TestPlanIssueConstants(t *testing.T) {
	t.Parallel()

	if PlanIssueSeverityError != "error" {
		t.Fatalf("PlanIssueSeverityError = %q, want error", PlanIssueSeverityError)
	}
	if PlanIssueSeverityWarning != "warning" {
		t.Fatalf("PlanIssueSeverityWarning = %q, want warning", PlanIssueSeverityWarning)
	}
	if PlanIssueTargetPathConflict != "target_path_conflict" {
		t.Fatalf("PlanIssueTargetPathConflict = %q, want target_path_conflict", PlanIssueTargetPathConflict)
	}
}

func TestApplyOperationStatusConstants(t *testing.T) {
	t.Parallel()

	if ApplyOperationStatusCompleted != "completed" {
		t.Fatalf("ApplyOperationStatusCompleted = %q, want completed", ApplyOperationStatusCompleted)
	}
	if ApplyOperationStatusFailed != "failed" {
		t.Fatalf("ApplyOperationStatusFailed = %q, want failed", ApplyOperationStatusFailed)
	}
	if ApplyOperationStatusSkipped != "skipped" {
		t.Fatalf("ApplyOperationStatusSkipped = %q, want skipped", ApplyOperationStatusSkipped)
	}
}

func TestOperationPlanSupportsMixedOperations(t *testing.T) {
	t.Parallel()

	sourcePath := "/mods/skyui/Data/SkyUI.esp"
	backupPath := "/backups/Skyrim/Data/SkyUI.esp.bak"
	targetPath := "/games/Skyrim/Data/Existing.esp"

	plan := OperationPlan{
		Operations: []Operation{
			{
				Type:       OperationTypeCopy,
				SourcePath: &sourcePath,
				TargetPath: "/games/Skyrim/Data/SkyUI.esp",
				BackupPath: &backupPath,
				Conflict:   false,
				Mod: ModContext{
					ModID:   1,
					ModName: "SkyUI",
				},
			},
			{
				Type:       OperationTypeReplace,
				TargetPath: targetPath,
				Conflict:   true,
				Mod: ModContext{
					ModID:   2,
					ModName: "Override Pack",
				},
			},
			{
				Type:       OperationTypeCreateDirectory,
				TargetPath: "/games/Skyrim/Data/Interface",
				Mod: ModContext{
					ModID:   3,
					ModName: "Interface Files",
				},
			},
		},
		Issues: []PlanIssue{
			{
				Severity:   PlanIssueSeverityWarning,
				Kind:       PlanIssueReplaceExistingTarget,
				Message:    "replace warning",
				ProfileID:  1,
				TargetPath: &targetPath,
			},
		},
		CanApply: true,
	}

	if len(plan.Operations) != 3 {
		t.Fatalf("len(plan.Operations) = %d, want 3", len(plan.Operations))
	}
	if plan.Operations[0].Type != OperationTypeCopy || plan.Operations[1].Type != OperationTypeReplace || plan.Operations[2].Type != OperationTypeCreateDirectory {
		t.Fatalf("plan.Operations types = %+v, want mixed future-safe operation types", plan.Operations)
	}
	if len(plan.Issues) != 1 || !plan.CanApply {
		t.Fatalf("plan metadata = %+v, want one warning issue and CanApply=true", plan)
	}
}

func TestOperationPreservesOptionalPathsAndContext(t *testing.T) {
	t.Parallel()

	sourcePath := "/mods/skyui/Data/SkyUI.esp"
	backupPath := "/backups/Skyrim/Data/SkyUI.esp.bak"

	operation := Operation{
		Type:       OperationTypeCopy,
		SourcePath: &sourcePath,
		TargetPath: "/games/Skyrim/Data/SkyUI.esp",
		BackupPath: &backupPath,
		Conflict:   true,
		Mod: ModContext{
			ModID:   42,
			ModName: "SkyUI",
		},
	}

	if operation.SourcePath == nil || *operation.SourcePath != sourcePath {
		t.Fatalf("operation.SourcePath = %#v, want %q", operation.SourcePath, sourcePath)
	}
	if operation.BackupPath == nil || *operation.BackupPath != backupPath {
		t.Fatalf("operation.BackupPath = %#v, want %q", operation.BackupPath, backupPath)
	}
	if !operation.Conflict {
		t.Fatal("operation.Conflict = false, want true")
	}
	if operation.Mod.ModID != 42 || operation.Mod.ModName != "SkyUI" {
		t.Fatalf("operation.Mod = %+v, want owning mod context", operation.Mod)
	}
}

func TestOperationAllowsUnsetOptionalPaths(t *testing.T) {
	t.Parallel()

	operation := Operation{
		Type:       OperationTypeCreateDirectory,
		TargetPath: "/games/Skyrim/Data/Interface",
		Mod: ModContext{
			ModID:   7,
			ModName: "Interface Files",
		},
	}

	if operation.SourcePath != nil {
		t.Fatalf("operation.SourcePath = %#v, want nil", operation.SourcePath)
	}
	if operation.BackupPath != nil {
		t.Fatalf("operation.BackupPath = %#v, want nil", operation.BackupPath)
	}
}

func TestOperationPlanUsesPlainExportedStructs(t *testing.T) {
	t.Parallel()

	planType := reflect.TypeOf(OperationPlan{})
	if planType.Kind() != reflect.Struct {
		t.Fatalf("OperationPlan kind = %v, want struct", planType.Kind())
	}
	for _, name := range []string{"Operations", "Issues", "CanApply"} {
		field, ok := planType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("OperationPlan field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("OperationPlan field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	operationType := reflect.TypeOf(Operation{})
	wantFields := []string{"Type", "SourcePath", "TargetPath", "BackupPath", "Conflict", "Mod"}
	for _, name := range wantFields {
		field, ok := operationType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("Operation field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("Operation field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	issueType := reflect.TypeOf(PlanIssue{})
	for _, name := range []string{"Severity", "Kind", "Message", "ProfileID", "SourcePath", "TargetPath", "Mod"} {
		field, ok := issueType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("PlanIssue field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("PlanIssue field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	modType := reflect.TypeOf(ModContext{})
	for _, name := range []string{"ModID", "ModName"} {
		field, ok := modType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("ModContext field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("ModContext field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	resultType := reflect.TypeOf(ApplyOperationPlanResult{})
	for _, name := range []string{"Success", "CompletedCount", "FailedCount", "SkippedCount", "Results", "Manifest"} {
		field, ok := resultType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("ApplyOperationPlanResult field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("ApplyOperationPlanResult field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	operationResultType := reflect.TypeOf(ApplyOperationResult{})
	for _, name := range []string{"OperationIndex", "Operation", "Status", "Message", "Error"} {
		field, ok := operationResultType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("ApplyOperationResult field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("ApplyOperationResult field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	manifestType := reflect.TypeOf(AppliedOperationManifest{})
	for _, name := range []string{"AddedFiles", "ReplacedFiles", "CreatedDirectories"} {
		field, ok := manifestType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("AppliedOperationManifest field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("AppliedOperationManifest field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	addedFileType := reflect.TypeOf(AppliedFileManifestEntry{})
	for _, name := range []string{"OperationIndex", "Mod", "SourcePath", "TargetPath", "SHA256", "SizeBytes"} {
		field, ok := addedFileType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("AppliedFileManifestEntry field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("AppliedFileManifestEntry field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	replacedFileType := reflect.TypeOf(ReplacedFileManifestEntry{})
	for _, name := range []string{"OperationIndex", "Mod", "SourcePath", "TargetPath", "SHA256", "SizeBytes", "BackupPath", "BackupSHA256", "BackupSizeBytes"} {
		field, ok := replacedFileType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("ReplacedFileManifestEntry field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("ReplacedFileManifestEntry field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}

	directoryType := reflect.TypeOf(AppliedDirectoryManifestEntry{})
	for _, name := range []string{"OperationIndex", "Mod", "TargetPath"} {
		field, ok := directoryType.FieldByName(name)
		if !ok || !field.IsExported() {
			t.Fatalf("AppliedDirectoryManifestEntry field %q exported = %v, want true", name, ok && field.IsExported())
		}
		if field.Tag != "" {
			t.Fatalf("AppliedDirectoryManifestEntry field %q tag = %q, want empty tag for plain Wails binding", name, string(field.Tag))
		}
	}
}
