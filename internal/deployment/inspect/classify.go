package inspect

import (
	"path/filepath"
	"strings"
)

type fileClass string

const (
	fileClassText    fileClass = "text"
	fileClassPE      fileClass = "pe"
	fileClassImage   fileClass = "image"
	fileClassArchive fileClass = "archive"
	fileClassBinary  fileClass = "binary"
)

var textExtensions = map[string]struct{}{
	".cfg":        {},
	".conf":       {},
	".ini":        {},
	".json":       {},
	".log":        {},
	".lua":        {},
	".md":         {},
	".properties": {},
	".toml":       {},
	".txt":        {},
	".xml":        {},
	".yaml":       {},
	".yml":        {},
}

var imageExtensions = map[string]struct{}{
	".gif":  {},
	".jpeg": {},
	".jpg":  {},
	".png":  {},
}

var archiveExtensions = map[string]struct{}{
	".7z":      {},
	".bz2":     {},
	".gz":      {},
	".rar":     {},
	".tar":     {},
	".tar.bz2": {},
	".tar.gz":  {},
	".tar.xz":  {},
	".tar.zst": {},
	".tbz2":    {},
	".tgz":     {},
	".txz":     {},
	".tzst":    {},
	".xz":      {},
	".zip":     {},
	".zst":     {},
}

func classifyRelativePath(relativePath string) fileClass {
	lowerName := strings.ToLower(filepath.Base(relativePath))

	for suffix := range archiveExtensions {
		if strings.HasSuffix(lowerName, suffix) {
			return fileClassArchive
		}
	}

	extension := strings.ToLower(filepath.Ext(lowerName))
	switch extension {
	case ".dll", ".exe":
		return fileClassPE
	}

	if _, found := imageExtensions[extension]; found {
		return fileClassImage
	}
	if _, found := textExtensions[extension]; found {
		return fileClassText
	}

	return fileClassBinary
}

func inspectionKindForClass(class fileClass) InspectionKind {
	switch class {
	case fileClassText:
		return InspectionTextDiff
	case fileClassPE:
		return InspectionPEMetadata
	case fileClassImage:
		return InspectionImageMetadata
	case fileClassArchive:
		return InspectionArchiveListing
	default:
		return InspectionBinaryFallback
	}
}
