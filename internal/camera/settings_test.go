package camera

import "testing"

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	if s.Width != DefaultWidth {
		t.Errorf("Width = %d, want %d", s.Width, DefaultWidth)
	}
	if s.Height != DefaultHeight {
		t.Errorf("Height = %d, want %d", s.Height, DefaultHeight)
	}
	if s.FPS != DefaultFPS {
		t.Errorf("FPS = %d, want %d", s.FPS, DefaultFPS)
	}
	if s.Format != DefaultFormat {
		t.Errorf("Format = %q, want %q", s.Format, DefaultFormat)
	}
	if s.MaxCameras != DefaultMaxCameras {
		t.Errorf("MaxCameras = %d, want %d", s.MaxCameras, DefaultMaxCameras)
	}
}

func TestNewManagerWithSettings_DefaultMaxCameras(t *testing.T) {
	m := NewManagerWithSettings(Settings{
		Width:  640,
		Height: 480,
		FPS:    20,
		Format: "mjpeg",
	}, true)
	s := m.GetSettings()
	if s.MaxCameras != DefaultMaxCameras {
		t.Errorf("MaxCameras = %d, want %d", s.MaxCameras, DefaultMaxCameras)
	}
}
