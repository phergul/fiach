//go:build windows

package reshade

import (
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	winTrustUIChoiceNone              = 2
	winTrustRevocationChecksNone      = 0
	winTrustChoiceFile                = 1
	winTrustProviderFlagCacheOnlyURLs = 0x00001000
	cryptMessageSignerInfoParameter   = 6

	trustStatusSuccess       = uint32(0)
	trustStatusNoSignature   = uint32(0x800B0100)
	trustStatusUntrustedRoot = uint32(0x800B0109)
)

var (
	winTrustDLL                   = windows.NewLazySystemDLL("wintrust.dll")
	winVerifyTrustProcedure       = winTrustDLL.NewProc("WinVerifyTrust")
	crypt32DLL                    = windows.NewLazySystemDLL("crypt32.dll")
	cryptMessageGetParamProcedure = crypt32DLL.NewProc("CryptMsgGetParam")
	cryptMessageCloseProcedure    = crypt32DLL.NewProc("CryptMsgClose")
	genericVerifyV2Action         = windows.GUID{
		Data1: 0x00AAC56B,
		Data2: 0xCD44,
		Data3: 0x11D0,
		Data4: [8]byte{0x8C, 0xC2, 0x00, 0xC0, 0x4F, 0xC2, 0x95, 0xEE},
	}
	emailAddressOID = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}
)

type platformInstallerSignatureVerifier struct{}

type winTrustFileInfo struct {
	Size         uint32
	FilePath     *uint16
	File         windows.Handle
	KnownSubject *windows.GUID
}

type winTrustData struct {
	Size               uint32
	PolicyCallbackData uintptr
	SIPClientData      uintptr
	UIChoice           uint32
	RevocationChecks   uint32
	UnionChoice        uint32
	FileInfo           *winTrustFileInfo
	StateAction        uint32
	StateData          windows.Handle
	URLReference       *uint16
	ProviderFlags      uint32
	UIContext          uint32
	SignatureSettings  uintptr
}

type cryptMessageSignerInfo struct {
	Version                   uint32
	Issuer                    windows.CertNameBlob
	SerialNumber              windows.CryptIntegerBlob
	HashAlgorithm             windows.CryptAlgorithmIdentifier
	HashEncryptionAlgorithm   windows.CryptAlgorithmIdentifier
	EncryptedHash             windows.DataBlob
	AuthenticatedAttributes   cryptAttributes
	UnauthenticatedAttributes cryptAttributes
}

type cryptAttributes struct {
	Count      uint32
	Attributes uintptr
}

func (platformInstallerSignatureVerifier) VerifyInstallerSignature(
	path string,
	variant InstallerVariant,
) (signature InstallerSignature, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("verify ReShade installer signature: %w", err)
		}
	}()
	pathPointer, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return InstallerSignature{}, err
	}
	fileInfo := winTrustFileInfo{
		Size:     uint32(unsafe.Sizeof(winTrustFileInfo{})),
		FilePath: pathPointer,
	}
	trustData := winTrustData{
		Size:             uint32(unsafe.Sizeof(winTrustData{})),
		UIChoice:         winTrustUIChoiceNone,
		RevocationChecks: winTrustRevocationChecksNone,
		UnionChoice:      winTrustChoiceFile,
		FileInfo:         &fileInfo,
		ProviderFlags:    winTrustProviderFlagCacheOnlyURLs,
	}
	status := callWinVerifyTrust(&trustData)

	if variant == InstallerVariantAddon {
		authenticodeStatus := trustStatusName(status)
		if status == trustStatusNoSignature {
			authenticodeStatus = "NotSigned"
			return enforceInstallerSignaturePolicy(
				variant,
				authenticodeStatus,
				"",
				"",
				"",
				"",
			)
		}
		if status != trustStatusSuccess && status != trustStatusUntrustedRoot {
			return enforceInstallerSignaturePolicy(
				variant,
				authenticodeStatus,
				"",
				"",
				"",
				"",
			)
		}
		certificate, err := signerCertificateFromFile(pathPointer)
		if err != nil {
			return InstallerSignature{}, err
		}
		subject := canonicalSignerSubject(certificate)
		spkiDigest := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
		certificateDigest := sha1.Sum(certificate.Raw)
		return enforceInstallerSignaturePolicy(
			variant,
			map[bool]string{true: "Valid", false: "UnknownError"}[status == trustStatusSuccess],
			trustStatusName(status),
			subject,
			hex.EncodeToString(spkiDigest[:]),
			hex.EncodeToString(certificateDigest[:]),
		)
	}
	if variant != InstallerVariantStandard {
		return InstallerSignature{}, fmt.Errorf("installer variant %q is invalid", variant)
	}
	if status != trustStatusSuccess && status != trustStatusUntrustedRoot {
		return InstallerSignature{}, fmt.Errorf(
			"standard installer Authenticode status is %s", trustStatusName(status))
	}
	certificate, err := signerCertificateFromFile(pathPointer)
	if err != nil {
		return InstallerSignature{}, err
	}
	subject := canonicalSignerSubject(certificate)
	spkiDigest := sha256.Sum256(certificate.RawSubjectPublicKeyInfo)
	certificateDigest := sha1.Sum(certificate.Raw)
	return enforceInstallerSignaturePolicy(
		variant,
		map[bool]string{true: "Valid", false: "UnknownError"}[status == trustStatusSuccess],
		trustStatusName(status),
		subject,
		hex.EncodeToString(spkiDigest[:]),
		hex.EncodeToString(certificateDigest[:]),
	)
}

func canonicalSignerSubject(certificate *x509.Certificate) string {
	emailAddress := ""
	for _, name := range certificate.Subject.Names {
		if name.Type.Equal(emailAddressOID) {
			emailAddress, _ = name.Value.(string)
			break
		}
	}
	if emailAddress == "" {
		return certificate.Subject.String()
	}
	return fmt.Sprintf("CN=%s, E=%s", certificate.Subject.CommonName, emailAddress)
}

func callWinVerifyTrust(data *winTrustData) uint32 {
	result, _, _ := winVerifyTrustProcedure.Call(
		0,
		uintptr(unsafe.Pointer(&genericVerifyV2Action)),
		uintptr(unsafe.Pointer(data)),
	)
	return uint32(result)
}

func signerCertificateFromFile(path *uint16) (*x509.Certificate, error) {
	var (
		encodingType uint32
		contentType  uint32
		formatType   uint32
		store        windows.Handle
		message      windows.Handle
		context      unsafe.Pointer
	)
	err := windows.CryptQueryObject(
		windows.CERT_QUERY_OBJECT_FILE,
		unsafe.Pointer(path),
		windows.CERT_QUERY_CONTENT_FLAG_PKCS7_SIGNED_EMBED,
		windows.CERT_QUERY_FORMAT_FLAG_BINARY,
		0,
		&encodingType,
		&contentType,
		&formatType,
		&store,
		&message,
		&context,
	)
	if err != nil {
		return nil, fmt.Errorf("query embedded Authenticode signature: %w", err)
	}
	defer windows.CertCloseStore(store, 0)
	defer cryptMessageCloseProcedure.Call(uintptr(message))

	var signerInfoSize uint32
	ok, _, callErr := cryptMessageGetParamProcedure.Call(
		uintptr(message),
		cryptMessageSignerInfoParameter,
		0,
		0,
		uintptr(unsafe.Pointer(&signerInfoSize)),
	)
	if ok == 0 {
		return nil, fmt.Errorf("read Authenticode signer information size: %w", callErr)
	}
	if signerInfoSize == 0 {
		return nil, fmt.Errorf("Authenticode signer information is empty")
	}
	signerInfoBuffer := make([]byte, signerInfoSize)
	ok, _, callErr = cryptMessageGetParamProcedure.Call(
		uintptr(message),
		cryptMessageSignerInfoParameter,
		0,
		uintptr(unsafe.Pointer(&signerInfoBuffer[0])),
		uintptr(unsafe.Pointer(&signerInfoSize)),
	)
	if ok == 0 {
		return nil, fmt.Errorf("read Authenticode signer information: %w", callErr)
	}
	signerInfo := (*cryptMessageSignerInfo)(unsafe.Pointer(&signerInfoBuffer[0]))
	certificateInfo := windows.CertInfo{
		SerialNumber: signerInfo.SerialNumber,
		Issuer:       signerInfo.Issuer,
	}
	certificateContext, err := windows.CertFindCertificateInStore(
		store,
		windows.X509_ASN_ENCODING|windows.PKCS_7_ASN_ENCODING,
		0,
		windows.CERT_FIND_SUBJECT_CERT,
		unsafe.Pointer(&certificateInfo),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("find Authenticode signer certificate: %w", err)
	}
	defer windows.CertFreeCertificateContext(certificateContext)
	if certificateContext.EncodedCert == nil || certificateContext.Length == 0 {
		return nil, fmt.Errorf("Authenticode signer certificate is missing")
	}
	encoded := unsafe.Slice(
		certificateContext.EncodedCert,
		certificateContext.Length,
	)
	certificate, err := x509.ParseCertificate(encoded)
	if err != nil {
		return nil, fmt.Errorf("parse WinVerifyTrust signer certificate: %w", err)
	}
	return certificate, nil
}

func trustStatusName(status uint32) string {
	switch status {
	case trustStatusSuccess:
		return "success"
	case trustStatusNoSignature:
		return "TRUST_E_NOSIGNATURE"
	case trustStatusUntrustedRoot:
		return "CERT_E_UNTRUSTEDROOT"
	default:
		return fmt.Sprintf("%s (%#x)", syscall.Errno(status), status)
	}
}
