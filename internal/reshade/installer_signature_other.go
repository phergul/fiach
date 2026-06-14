//go:build !windows

package reshade

import "fmt"

type platformInstallerSignatureVerifier struct{}

func (platformInstallerSignatureVerifier) VerifyInstallerSignature(
	string,
	InstallerVariant,
) (InstallerSignature, error) {
	return InstallerSignature{}, fmt.Errorf("verify ReShade installer signature: unsupported platform")
}
