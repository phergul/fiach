package reshade

import "golang.org/x/mod/semver"

func semverIsCanonicalStable(version string) bool {
	return semver.IsValid(version) &&
		semver.Canonical(version) == version &&
		semver.Prerelease(version) == ""
}
