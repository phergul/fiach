//go:build production

package appmode

func IsDev() bool {
	return false
}

func DataDirName() string {
	return "fiach"
}
