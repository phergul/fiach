package reshade

import (
	"strings"
	"testing"
)

func TestEnforceInstallerSignaturePolicyAcceptsPinnedSelfSignedStandardBuild(t *testing.T) {
	t.Parallel()

	signature, err := enforceInstallerSignaturePolicy(
		InstallerVariantStandard,
		"UnknownError",
		"self-signed certificate",
		reShadeSignerSubject,
		reShadeSignerSPKISHA256,
		"589690208A5E52FB96980C4A6698F50ACD47C49F",
	)
	if err != nil {
		t.Fatalf("enforceInstallerSignaturePolicy() error = %v", err)
	}
	if signature.Status != InstallerSignatureStatusVerified ||
		signature.SPKISHA256 != reShadeSignerSPKISHA256 ||
		signature.CertificateSHA1 != "589690208a5e52fb96980c4a6698f50acd47c49f" {
		t.Fatalf("signature = %+v", signature)
	}
}

func TestEnforceInstallerSignaturePolicyRejectsUntrustedStandardBuilds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  string
		subject string
		key     string
		want    string
	}{
		{
			name:    "altered file",
			status:  "HashMismatch",
			subject: reShadeSignerSubject,
			key:     reShadeSignerSPKISHA256,
			want:    "HashMismatch",
		},
		{
			name:    "unsigned",
			status:  "NotSigned",
			subject: "",
			key:     "",
			want:    "NotSigned",
		},
		{
			name:    "wrong subject",
			status:  "Valid",
			subject: "CN=Other",
			key:     reShadeSignerSPKISHA256,
			want:    "subject",
		},
		{
			name:    "wrong key",
			status:  "Valid",
			subject: reShadeSignerSubject,
			key:     strings.Repeat("0", 64),
			want:    "public key",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := enforceInstallerSignaturePolicy(
				InstallerVariantStandard,
				test.status,
				"",
				test.subject,
				test.key,
				"",
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestEnforceInstallerSignaturePolicyAcceptsUnsignedAddonBuild(t *testing.T) {
	t.Parallel()

	signature, err := enforceInstallerSignaturePolicy(
		InstallerVariantAddon,
		"NotSigned",
		"",
		"",
		"",
		"",
	)
	if err != nil || signature.Status != InstallerSignatureStatusUnsigned {
		t.Fatalf("signature = %+v, error = %v", signature, err)
	}
}

func TestEnforceInstallerSignaturePolicyAcceptsPinnedSelfSignedAddonBuild(t *testing.T) {
	t.Parallel()

	signature, err := enforceInstallerSignaturePolicy(
		InstallerVariantAddon,
		"UnknownError",
		"self-signed certificate",
		reShadeSignerSubject,
		reShadeSignerSPKISHA256,
		"589690208A5E52FB96980C4A6698F50ACD47C49F",
	)
	if err != nil {
		t.Fatalf("enforceInstallerSignaturePolicy() error = %v", err)
	}
	if signature.Status != InstallerSignatureStatusVerified ||
		signature.SPKISHA256 != reShadeSignerSPKISHA256 ||
		signature.CertificateSHA1 != "589690208a5e52fb96980c4a6698f50acd47c49f" {
		t.Fatalf("signature = %+v", signature)
	}
}

func TestEnforceInstallerSignaturePolicyRejectsUntrustedAddonBuilds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  string
		subject string
		key     string
		want    string
	}{
		{
			name:    "altered file",
			status:  "HashMismatch",
			subject: reShadeSignerSubject,
			key:     reShadeSignerSPKISHA256,
			want:    "signature state changed",
		},
		{
			name:    "wrong subject",
			status:  "UnknownError",
			subject: "CN=Other",
			key:     reShadeSignerSPKISHA256,
			want:    "subject",
		},
		{
			name:    "wrong key",
			status:  "Valid",
			subject: reShadeSignerSubject,
			key:     strings.Repeat("0", 64),
			want:    "public key",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := enforceInstallerSignaturePolicy(
				InstallerVariantAddon,
				test.status,
				"",
				test.subject,
				test.key,
				"",
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
