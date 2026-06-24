package thirdparty

import "testing"

func TestIsOptiScalerFinalAsset(t *testing.T) {
	t.Parallel()

	if !IsOptiScalerFinalAsset("Optiscaler_0.9.3-final.20260618.7z") {
		t.Fatal("IsOptiScalerFinalAsset(final archive) = false, want true")
	}
	if IsOptiScalerFinalAsset("Optiscaler_0.9.3-preview.7z") {
		t.Fatal("IsOptiScalerFinalAsset(preview archive) = true, want false")
	}
}
