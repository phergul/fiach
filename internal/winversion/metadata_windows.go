//go:build windows

package winversion

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	versionDLL              = windows.NewLazySystemDLL("version.dll")
	getFileVersionInfoSizeW = versionDLL.NewProc("GetFileVersionInfoSizeW")
	getFileVersionInfoW     = versionDLL.NewProc("GetFileVersionInfoW")
	verQueryValueW          = versionDLL.NewProc("VerQueryValueW")
)

func Read(path string) (metadata Metadata, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("read Windows version-resource metadata: %w", err)
		}
	}()

	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return Metadata{}, err
	}
	size, _, callErr := getFileVersionInfoSizeW.Call(uintptr(unsafe.Pointer(pathPointer)), 0)
	if size == 0 {
		if callErr != syscall.Errno(0) {
			return Metadata{}, callErr
		}
		return Metadata{}, errors.New("file has no readable version resource")
	}
	buffer := make([]byte, size)
	ok, _, callErr := getFileVersionInfoW.Call(
		uintptr(unsafe.Pointer(pathPointer)), 0, size, uintptr(unsafe.Pointer(&buffer[0])),
	)
	if ok == 0 {
		return Metadata{}, callErr
	}

	languages := versionLanguages(buffer)
	if len(languages) == 0 {
		languages = []string{"040904b0", "040904e4"}
	}
	for _, language := range languages {
		if metadata.CompanyName == "" {
			metadata.CompanyName = queryVersionString(buffer, language, "CompanyName")
		}
		if metadata.FileDescription == "" {
			metadata.FileDescription = queryVersionString(buffer, language, "FileDescription")
		}
		if metadata.FileVersion == "" {
			metadata.FileVersion = queryVersionString(buffer, language, "FileVersion")
		}
		if metadata.InternalName == "" {
			metadata.InternalName = queryVersionString(buffer, language, "InternalName")
		}
		if metadata.OriginalFilename == "" {
			metadata.OriginalFilename = queryVersionString(buffer, language, "OriginalFilename")
		}
		if metadata.ProductName == "" {
			metadata.ProductName = queryVersionString(buffer, language, "ProductName")
		}
		if metadata.ProductVersion == "" {
			metadata.ProductVersion = queryVersionString(buffer, language, "ProductVersion")
		}
	}
	return metadata, nil
}

func versionLanguages(buffer []byte) []string {
	pointer, length, ok := queryVersionValue(buffer, `\VarFileInfo\Translation`)
	if !ok || length < 4 {
		return nil
	}
	values := unsafe.Slice((*uint16)(pointer), int(length/2))
	languages := make([]string, 0, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		languages = append(languages, fmt.Sprintf("%04x%04x", values[index], values[index+1]))
	}
	return languages
}

func queryVersionString(buffer []byte, language string, field string) string {
	pointer, length, ok := queryVersionValue(buffer, `\StringFileInfo\`+language+`\`+field)
	if !ok || length == 0 {
		return ""
	}
	return windows.UTF16PtrToString((*uint16)(pointer))
}

func queryVersionValue(buffer []byte, query string) (unsafe.Pointer, uint32, bool) {
	queryPointer, err := syscall.UTF16PtrFromString(query)
	if err != nil {
		return nil, 0, false
	}
	var valuePointer unsafe.Pointer
	var length uint32
	ok, _, _ := verQueryValueW.Call(
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(unsafe.Pointer(queryPointer)),
		uintptr(unsafe.Pointer(&valuePointer)),
		uintptr(unsafe.Pointer(&length)),
	)
	return valuePointer, length, ok != 0
}
