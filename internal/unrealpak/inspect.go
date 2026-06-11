package unrealpak

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/phergul/fiach/internal/fileignore"
	"github.com/phergul/fiach/internal/fileops"
)

var recognizedExtensions = map[string]struct{}{
	".pak":  {},
	".ucas": {},
	".utoc": {},
}

type File struct {
	SourcePath string
	Name       string
	SizeBytes  int64
}

type Inspection struct {
	Files     []File
	SizeBytes int64
	Warnings  []string
}

type packageGroup struct {
	stem  string
	files map[string]File
}

func Inspect(sourceRoot string) (inspection Inspection, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("inspect Unreal package source: %w", err)
		}
	}()

	groups := map[string]*packageGroup{}
	targetNames := map[string]string{}
	ignoredFileCount := 0

	err = filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if sourcePath == sourceRoot {
			return nil
		}
		if fileignore.Has(entry.Name()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}

		info, infoErr := fileops.ValidateDirEntryIsRegularFile("source file", entry)
		if infoErr != nil {
			return infoErr
		}

		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if _, recognized := recognizedExtensions[extension]; !recognized {
			ignoredFileCount++
			return nil
		}

		nameKey := strings.ToLower(entry.Name())
		if previousPath, exists := targetNames[nameKey]; exists {
			return fmt.Errorf(
				"files %q and %q both flatten to target name %q",
				previousPath,
				sourcePath,
				entry.Name(),
			)
		}
		targetNames[nameKey] = sourcePath

		stem := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		groupKey := strings.ToLower(stem)
		group := groups[groupKey]
		if group == nil {
			group = &packageGroup{
				stem:  stem,
				files: map[string]File{},
			}
			groups[groupKey] = group
		} else if group.stem != stem {
			return fmt.Errorf(
				"package group files use inconsistent stem casing %q and %q",
				group.stem,
				stem,
			)
		}
		group.files[extension] = File{
			SourcePath: sourcePath,
			Name:       entry.Name(),
			SizeBytes:  info.Size(),
		}
		return nil
	})
	if err != nil {
		return Inspection{}, err
	}

	if len(groups) == 0 {
		return Inspection{}, errors.New("source contains no .pak, .ucas, or .utoc files")
	}

	groupKeys := make([]string, 0, len(groups))
	for key := range groups {
		groupKeys = append(groupKeys, key)
	}
	sort.Strings(groupKeys)

	inspection.Files = []File{}
	inspection.Warnings = []string{}
	for _, key := range groupKeys {
		group := groups[key]
		_, hasPak := group.files[".pak"]
		_, hasUcas := group.files[".ucas"]
		_, hasUtoc := group.files[".utoc"]

		if hasUcas != hasUtoc {
			missingExtension := ".utoc"
			if hasUtoc {
				missingExtension = ".ucas"
			}
			return Inspection{}, fmt.Errorf(
				"package group %q is incomplete: matching %s file is required",
				group.stem,
				missingExtension,
			)
		}
		if !hasPak && !hasUcas {
			return Inspection{}, fmt.Errorf("package group %q is incomplete", group.stem)
		}

		if !strings.HasSuffix(strings.ToLower(group.stem), "_p") {
			inspection.Warnings = append(
				inspection.Warnings,
				fmt.Sprintf("Package group %q does not use the _P suffix commonly required for patch priority.", group.stem),
			)
		}

		for _, extension := range []string{".pak", ".ucas", ".utoc"} {
			file, found := group.files[extension]
			if !found {
				continue
			}
			inspection.Files = append(inspection.Files, file)
			inspection.SizeBytes += file.SizeBytes
		}
	}

	sort.SliceStable(inspection.Files, func(i int, j int) bool {
		return strings.ToLower(inspection.Files[i].Name) < strings.ToLower(inspection.Files[j].Name)
	})
	if ignoredFileCount > 0 {
		inspection.Warnings = append(
			inspection.Warnings,
			fmt.Sprintf("Ignored %d unsupported source file(s); only .pak, .ucas, and .utoc files will be installed.", ignoredFileCount),
		)
	}

	return inspection, nil
}
