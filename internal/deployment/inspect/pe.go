package inspect

import (
	"debug/pe"
	"fmt"
	"strings"

	"github.com/phergul/fiach/internal/fileops"
)

func readPEMetadata(path string) (*PEMetadata, error) {
	info, err := fileops.StatRegularFile("PE file", path)
	if err != nil {
		return nil, err
	}

	sha256Hex, sizeBytes, err := fileops.FileIntegrity(path)
	if err != nil {
		return nil, fmt.Errorf("hash PE file %q: %w", path, err)
	}

	file, err := pe.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open PE file %q: %w", path, err)
	}
	defer file.Close()

	metadata := &PEMetadata{
		Machine:         peMachineName(file.FileHeader.Machine),
		SectionCount:    len(file.Sections),
		Characteristics: peCharacteristics(file.FileHeader.Characteristics),
		IsDLL:           file.FileHeader.Characteristics&pe.IMAGE_FILE_DLL != 0,
		IsEXE:           file.FileHeader.Characteristics&pe.IMAGE_FILE_DLL == 0,
		SHA256:          sha256Hex,
		SizeBytes:       sizeBytes,
	}

	if metadata.SizeBytes == 0 {
		metadata.SizeBytes = info.Size()
	}

	return metadata, nil
}

func peMachineName(machine uint16) string {
	switch machine {
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "AMD64"
	case pe.IMAGE_FILE_MACHINE_I386:
		return "I386"
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return "ARM64"
	default:
		return fmt.Sprintf("0x%04x", machine)
	}
}

func peCharacteristics(value uint16) string {
	flags := make([]string, 0, 4)
	if value&pe.IMAGE_FILE_EXECUTABLE_IMAGE != 0 {
		flags = append(flags, "executable")
	}
	if value&pe.IMAGE_FILE_DLL != 0 {
		flags = append(flags, "dll")
	}
	if value&pe.IMAGE_FILE_SYSTEM != 0 {
		flags = append(flags, "system")
	}
	if len(flags) == 0 {
		return "none"
	}

	return strings.Join(flags, ", ")
}
