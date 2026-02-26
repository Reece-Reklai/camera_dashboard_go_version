package ui

import (
	"image"
	"image/color"
	"testing"
)

func TestNightModeLUT_Values(t *testing.T) {
	// LUT[0] should be 0 (black stays black)
	if nightModeLUT[0] != 0 {
		t.Errorf("nightModeLUT[0] = %d, want 0", nightModeLUT[0])
	}

	// LUT[100] should be uint8(100 * 1.6) = 160
	expected100 := uint8(160)
	if nightModeLUT[100] != expected100 {
		t.Errorf("nightModeLUT[100] = %d, want %d", nightModeLUT[100], expected100)
	}

	// LUT[159] should be uint8(159 * 1.6) = 254
	v := float64(159) * 1.6
	expected159 := uint8(v)
	if nightModeLUT[159] != expected159 {
		t.Errorf("nightModeLUT[159] = %d, want %d", nightModeLUT[159], expected159)
	}

	// LUT[160] = 256 -> clamped to 255
	if nightModeLUT[160] != 255 {
		t.Errorf("nightModeLUT[160] = %d, want 255 (clamped)", nightModeLUT[160])
	}

	// LUT[255] should be clamped to 255
	if nightModeLUT[255] != 255 {
		t.Errorf("nightModeLUT[255] = %d, want 255", nightModeLUT[255])
	}

	// LUT should be monotonically non-decreasing
	for i := 1; i < 256; i++ {
		if nightModeLUT[i] < nightModeLUT[i-1] {
			t.Errorf("nightModeLUT not monotonic: LUT[%d]=%d < LUT[%d]=%d",
				i, nightModeLUT[i], i-1, nightModeLUT[i-1])
		}
	}
}

func TestApplyNightMode_RGBA(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	// Set a known pixel: pure white (255,255,255)
	src.Set(0, 0, color.RGBA{255, 255, 255, 255})
	// Set a known pixel: pure red (255,0,0)
	src.Set(1, 0, color.RGBA{255, 0, 0, 255})
	// Set a known pixel: pure green (0,255,0)
	src.Set(0, 1, color.RGBA{0, 255, 0, 255})
	// Set a known pixel: black (0,0,0)
	src.Set(1, 1, color.RGBA{0, 0, 0, 255})

	dst := applyNightMode(src)

	// White: gray = (299*255 + 587*255 + 114*255)/1000 = 255
	// boosted = nightModeLUT[255] = 255
	r, g, b, a := dst.At(0, 0).RGBA()
	if uint8(r>>8) != 255 || uint8(g>>8) != 0 || uint8(b>>8) != 0 || uint8(a>>8) != 255 {
		t.Errorf("white pixel: got RGBA(%d,%d,%d,%d), want (255,0,0,255)",
			uint8(r>>8), uint8(g>>8), uint8(b>>8), uint8(a>>8))
	}

	// Black: gray = 0, boosted = 0
	r, g, b, _ = dst.At(1, 1).RGBA()
	if uint8(r>>8) != 0 || uint8(g>>8) != 0 || uint8(b>>8) != 0 {
		t.Errorf("black pixel: got RGB(%d,%d,%d), want (0,0,0)",
			uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}

	// Red: gray = 299*255/1000 = 76, boosted = nightModeLUT[76]
	expectedRed := nightModeLUT[76]
	r, _, _, _ = dst.At(1, 0).RGBA()
	if uint8(r>>8) != expectedRed {
		t.Errorf("red pixel: got R=%d, want %d", uint8(r>>8), expectedRed)
	}

	// Green: gray = 587*255/1000 = 149, boosted = nightModeLUT[149]
	expectedGreen := nightModeLUT[149]
	r, _, _, _ = dst.At(0, 1).RGBA()
	if uint8(r>>8) != expectedGreen {
		t.Errorf("green pixel: got R=%d, want %d", uint8(r>>8), expectedGreen)
	}

	// All output pixels should have G=0, B=0
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			_, g, b, _ := dst.At(x, y).RGBA()
			if uint8(g>>8) != 0 || uint8(b>>8) != 0 {
				t.Errorf("pixel (%d,%d): G=%d B=%d, want G=0 B=0", x, y, uint8(g>>8), uint8(b>>8))
			}
		}
	}
}

func TestApplyNightMode_NRGBA(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.NRGBA{255, 255, 255, 255})
	src.Set(1, 0, color.NRGBA{0, 0, 0, 255})

	dst := applyNightMode(src)

	// White -> should be red 255
	r, g, b, _ := dst.At(0, 0).RGBA()
	if uint8(r>>8) != 255 || uint8(g>>8) != 0 || uint8(b>>8) != 0 {
		t.Errorf("NRGBA white: got RGB(%d,%d,%d), want (255,0,0)",
			uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}

	// Black -> should be 0
	r, _, _, _ = dst.At(1, 0).RGBA()
	if uint8(r>>8) != 0 {
		t.Errorf("NRGBA black: got R=%d, want 0", uint8(r>>8))
	}
}

func TestApplyNightMode_GenericImage(t *testing.T) {
	// Use image.YCbCr to trigger the generic fallback path
	src := image.NewGray(image.Rect(0, 0, 2, 1))
	src.SetGray(0, 0, color.Gray{Y: 128})
	src.SetGray(1, 0, color.Gray{Y: 0})

	dst := applyNightMode(src)

	// Gray 128: luminance = 128, boosted = nightModeLUT[128]
	expected := nightModeLUT[128]
	r, g, b, _ := dst.At(0, 0).RGBA()
	if uint8(r>>8) != expected || uint8(g>>8) != 0 || uint8(b>>8) != 0 {
		t.Errorf("gray 128: got RGB(%d,%d,%d), want (%d,0,0)",
			uint8(r>>8), uint8(g>>8), uint8(b>>8), expected)
	}
}

func TestApplyNightMode_OutputSize(t *testing.T) {
	src := image.NewRGBA(image.Rect(10, 20, 50, 60))
	dst := applyNightMode(src)

	if dst.Bounds().Dx() != 40 || dst.Bounds().Dy() != 40 {
		t.Errorf("output size = %dx%d, want 40x40", dst.Bounds().Dx(), dst.Bounds().Dy())
	}
}

func TestNightModeColor(t *testing.T) {
	// White -> gray=255 -> boosted=255 -> (255,0,0,255)
	result := nightModeColor(color.White)
	if result.R != 255 || result.G != 0 || result.B != 0 || result.A != 255 {
		t.Errorf("nightModeColor(white) = %v, want {255,0,0,255}", result)
	}

	// Black -> gray=0 -> boosted=0 -> (0,0,0,255)
	result = nightModeColor(color.Black)
	if result.R != 0 || result.G != 0 || result.B != 0 || result.A != 255 {
		t.Errorf("nightModeColor(black) = %v, want {0,0,0,255}", result)
	}

	// Mid-gray (128,128,128) -> gray=128 -> boosted=nightModeLUT[128]
	result = nightModeColor(color.RGBA{128, 128, 128, 255})
	expected := nightModeLUT[128]
	if result.R != expected || result.G != 0 || result.B != 0 {
		t.Errorf("nightModeColor(128,128,128) = R=%d, want R=%d, G=0, B=0", result.R, expected)
	}
}

func TestBrightnessLUTPresets(t *testing.T) {
	// 100% should be identity
	if brightnessLUTs[100][200] != 200 {
		t.Errorf("brightness 100%% LUT[200] = %d, want 200", brightnessLUTs[100][200])
	}

	// 150% should boost and clamp
	if brightnessLUTs[150][100] != 150 {
		t.Errorf("brightness 150%% LUT[100] = %d, want 150", brightnessLUTs[150][100])
	}
	if brightnessLUTs[150][200] != 255 {
		t.Errorf("brightness 150%% LUT[200] = %d, want 255", brightnessLUTs[150][200])
	}

	// 15% should darken strongly
	if brightnessLUTs[15][200] != 30 {
		t.Errorf("brightness 15%% LUT[200] = %d, want 30", brightnessLUTs[15][200])
	}
}

func TestApplyBrightnessPercentReuse(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 1))
	src.Set(0, 0, color.RGBA{100, 150, 200, 255})
	src.Set(1, 0, color.RGBA{255, 0, 10, 255})

	// Brighten
	dst := applyBrightnessPercentReuse(src, 150, nil)
	r, g, b, _ := dst.At(0, 0).RGBA()
	if uint8(r>>8) != 150 || uint8(g>>8) != 225 || uint8(b>>8) != 255 {
		t.Errorf("150%% pixel0 got (%d,%d,%d), want (150,225,255)",
			uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}

	// Darken and reuse buffer
	dst2 := applyBrightnessPercentReuse(src, 60, dst)
	if dst2 != dst {
		t.Error("expected destination buffer reuse")
	}
	r, g, b, _ = dst2.At(1, 0).RGBA()
	if uint8(r>>8) != 153 || uint8(g>>8) != 0 || uint8(b>>8) != 6 {
		t.Errorf("60%% pixel1 got (%d,%d,%d), want (153,0,6)",
			uint8(r>>8), uint8(g>>8), uint8(b>>8))
	}
}
