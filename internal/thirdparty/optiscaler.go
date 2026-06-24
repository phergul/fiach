package thirdparty

import "regexp"

var optiScalerFinalAssetName = regexp.MustCompile(`(?i)^optiscaler_.*final.*\.7z$`)

func IsOptiScalerFinalAsset(name string) bool {
	return optiScalerFinalAssetName.MatchString(name)
}
