//go:build !production

package appmode

func IsDev() bool {
	return true
}

func DataDirName() string {
	return "fiach-dev"
}
