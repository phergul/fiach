package reshade

import (
	"fmt"
	"strings"
)

const (
	reShadeSignerSubject    = "CN=ReShade, E=info@reshade.me"
	reShadeSignerSPKISHA256 = "4bc51f722d388da4048d5fbdee2dc68ceef699850ea3787471ef189e18589625"
)

func enforceInstallerSignaturePolicy(
	variant InstallerVariant,
	authenticodeStatus string,
	statusMessage string,
	subject string,
	spkiSHA256 string,
	certificateSHA1 string,
) (InstallerSignature, error) {
	if variant == InstallerVariantAddon {
		if authenticodeStatus != "NotSigned" || subject != "" || spkiSHA256 != "" {
			return InstallerSignature{}, fmt.Errorf(
				"full add-on installer signature state changed: status %q", authenticodeStatus)
		}
		return InstallerSignature{
			Status: InstallerSignatureStatusUnsigned,
		}, nil
	}
	if variant != InstallerVariantStandard {
		return InstallerSignature{}, fmt.Errorf("installer variant %q is invalid", variant)
	}
	if authenticodeStatus != "Valid" && authenticodeStatus != "UnknownError" {
		return InstallerSignature{}, fmt.Errorf(
			"standard installer Authenticode status is %q: %s",
			authenticodeStatus,
			strings.TrimSpace(statusMessage),
		)
	}
	if !strings.EqualFold(strings.TrimSpace(subject), reShadeSignerSubject) {
		return InstallerSignature{}, fmt.Errorf(
			"standard installer signer subject %q is not trusted", subject)
	}
	if !strings.EqualFold(strings.TrimSpace(spkiSHA256), reShadeSignerSPKISHA256) {
		return InstallerSignature{}, fmt.Errorf(
			"standard installer signer public key %q is not trusted", spkiSHA256)
	}
	return InstallerSignature{
		Status:          InstallerSignatureStatusVerified,
		Subject:         subject,
		SPKISHA256:      strings.ToLower(spkiSHA256),
		CertificateSHA1: strings.ToLower(certificateSHA1),
	}, nil
}
