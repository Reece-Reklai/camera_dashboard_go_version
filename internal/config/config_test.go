package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// DefaultConfig tests
// =============================================================================

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// Spot-check key defaults
	if cfg.LogLevel != "INFO" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "INFO")
	}
	if cfg.LogMaxBytes != 5*1024*1024 {
		t.Errorf("LogMaxBytes = %d, want %d", cfg.LogMaxBytes, 5*1024*1024)
	}
	if cfg.DynamicFPSEnabled != true {
		t.Errorf("DynamicFPSEnabled = %v, want true", cfg.DynamicFPSEnabled)
	}
	if cfg.CPULoadThreshold != 0.75 {
		t.Errorf("CPULoadThreshold = %f, want 0.75", cfg.CPULoadThreshold)
	}
	if cfg.CaptureWidth != 640 {
		t.Errorf("CaptureWidth = %d, want 640", cfg.CaptureWidth)
	}
	if cfg.CaptureHeight != 480 {
		t.Errorf("CaptureHeight = %d, want 480", cfg.CaptureHeight)
	}
	if cfg.CaptureFPS != 25 {
		t.Errorf("CaptureFPS = %d, want 25", cfg.CaptureFPS)
	}
	if cfg.UIFPS != 20 {
		t.Errorf("UIFPS = %d, want 20", cfg.UIFPS)
	}
	if cfg.HealthLogIntervalSec != 30.0 {
		t.Errorf("HealthLogIntervalSec = %f, want 30.0", cfg.HealthLogIntervalSec)
	}
	if cfg.KillDeviceHolders != true {
		t.Errorf("KillDeviceHolders = %v, want true", cfg.KillDeviceHolders)
	}
	if cfg.CameraSlotCount != 3 {
		t.Errorf("CameraSlotCount = %d, want 3", cfg.CameraSlotCount)
	}
	if cfg.StaleFrameTimeoutSec != 1.5 {
		t.Errorf("StaleFrameTimeoutSec = %f, want 1.5", cfg.StaleFrameTimeoutSec)
	}
	if cfg.MaxRestartsPerWindow != 3 {
		t.Errorf("MaxRestartsPerWindow = %d, want 3", cfg.MaxRestartsPerWindow)
	}
}

// =============================================================================
// asBool tests
// =============================================================================

func TestAsBool(t *testing.T) {
	tests := []struct {
		input    string
		fallback bool
		want     bool
	}{
		{"true", false, true},
		{"True", false, true},
		{"TRUE", false, true},
		{"1", false, true},
		{"yes", false, true},
		{"on", false, true},
		{"false", true, false},
		{"False", true, false},
		{"0", true, false},
		{"no", true, false},
		{"off", true, false},
		{"", true, true},     // empty -> fallback
		{"", false, false},   // empty -> fallback
		{"junk", true, true}, // unrecognised -> fallback
		{"junk", false, false},
		{"  true  ", false, true}, // whitespace trimmed
	}

	for _, tc := range tests {
		got := asBool(tc.input, tc.fallback)
		if got != tc.want {
			t.Errorf("asBool(%q, %v) = %v, want %v", tc.input, tc.fallback, got, tc.want)
		}
	}
}

// =============================================================================
// asInt tests
// =============================================================================

func TestAsInt(t *testing.T) {
	min5 := intPtr(5)
	max20 := intPtr(20)

	tests := []struct {
		input    string
		fallback int
		minVal   *int
		maxVal   *int
		want     int
	}{
		{"10", 0, nil, nil, 10},
		{"", 42, nil, nil, 42},                // empty -> fallback
		{"abc", 42, nil, nil, 42},             // parse error -> fallback
		{"3", 0, min5, nil, 5},                // clamped to min
		{"25", 0, nil, max20, 20},             // clamped to max
		{"10", 0, min5, max20, 10},            // within range
		{"  15  ", 0, nil, nil, 15},           // whitespace trimmed
		{"-5", 0, intPtr(0), nil, 0},          // negative clamped to min 0
		{"100", 0, intPtr(1), intPtr(60), 60}, // clamped to max 60
	}

	for _, tc := range tests {
		got := asInt(tc.input, tc.fallback, tc.minVal, tc.maxVal)
		if got != tc.want {
			t.Errorf("asInt(%q, %d, ...) = %d, want %d", tc.input, tc.fallback, got, tc.want)
		}
	}
}

// =============================================================================
// asFloat tests
// =============================================================================

func TestAsFloat(t *testing.T) {
	min0_1 := floatPtr(0.1)
	max1_0 := floatPtr(1.0)

	tests := []struct {
		input    string
		fallback float64
		minVal   *float64
		maxVal   *float64
		want     float64
	}{
		{"0.75", 0, nil, nil, 0.75},
		{"", 0.5, nil, nil, 0.5},              // empty -> fallback
		{"abc", 0.5, nil, nil, 0.5},           // parse error -> fallback
		{"0.05", 0, min0_1, nil, 0.1},         // clamped to min
		{"1.5", 0, nil, max1_0, 1.0},          // clamped to max
		{"0.5", 0, min0_1, max1_0, 0.5},       // within range
		{"  0.8  ", 0, nil, nil, 0.8},         // whitespace
		{"30.0", 0, floatPtr(5.0), nil, 30.0}, // within range, no max
		{"3.0", 0, floatPtr(5.0), nil, 5.0},   // clamped to min 5
	}

	for _, tc := range tests {
		got := asFloat(tc.input, tc.fallback, tc.minVal, tc.maxVal)
		if got != tc.want {
			t.Errorf("asFloat(%q, %f, ...) = %f, want %f", tc.input, tc.fallback, got, tc.want)
		}
	}
}

// =============================================================================
// parseINI tests
// =============================================================================

func TestParseINI(t *testing.T) {
	content := `# comment
[section1]
key1 = value1
key2 = value2

; another comment
[section2]
foo = bar
baz = qux
`
	tmp := writeTempFile(t, content)

	ini, err := parseINI(tmp)
	if err != nil {
		t.Fatalf("parseINI() error: %v", err)
	}

	if !ini.hasSection("section1") {
		t.Error("missing section1")
	}
	if !ini.hasSection("section2") {
		t.Error("missing section2")
	}
	if ini.hasSection("section3") {
		t.Error("unexpected section3")
	}

	val, ok := ini.get("section1", "key1")
	if !ok || val != "value1" {
		t.Errorf("section1.key1 = (%q, %v), want (%q, true)", val, ok, "value1")
	}

	val, ok = ini.get("section2", "foo")
	if !ok || val != "bar" {
		t.Errorf("section2.foo = (%q, %v), want (%q, true)", val, ok, "bar")
	}

	_, ok = ini.get("section1", "missing")
	if ok {
		t.Error("expected missing key to return ok=false")
	}
}

func TestParseINI_EmptyFile(t *testing.T) {
	tmp := writeTempFile(t, "")
	ini, err := parseINI(tmp)
	if err != nil {
		t.Fatalf("parseINI() error: %v", err)
	}
	if len(ini) != 0 {
		t.Errorf("expected empty iniData, got %d sections", len(ini))
	}
}

func TestParseINI_MissingFile(t *testing.T) {
	_, err := parseINI("/nonexistent/config.ini")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// =============================================================================
// Load tests
// =============================================================================

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("/nonexistent/config.ini")
	if err != nil {
		t.Fatalf("Load() returned error for missing file: %v", err)
	}
	// Should return defaults
	def := DefaultConfig()
	if cfg.CaptureFPS != def.CaptureFPS {
		t.Errorf("CaptureFPS = %d, want default %d", cfg.CaptureFPS, def.CaptureFPS)
	}
}

func TestLoad_FullINI(t *testing.T) {
	content := `
[logging]
level = DEBUG
file = /tmp/test.log
max_bytes = 1048576
backup_count = 5
stdout = false

[performance]
dynamic_fps = false
perf_check_interval_ms = 3000
cpu_load_threshold = 0.85
cpu_temp_threshold_c = 80.0
stress_hold_count = 5
recover_hold_count = 5
stale_frame_timeout_sec = 2.0
restart_cooldown_sec = 10.0
max_restarts_per_window = 5
restart_window_sec = 60.0

[camera]
rescan_interval_ms = 20000
failed_camera_cooldown_sec = 60.0
slot_count = 4
kill_device_holders = false

[profile]
capture_width = 1280
capture_height = 720
capture_fps = 30
ui_fps = 25

[health]
log_interval_sec = 60
`
	tmp := writeTempFile(t, content)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// Verify overridden values
	if cfg.LogLevel != "DEBUG" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "DEBUG")
	}
	if cfg.LogFile != "/tmp/test.log" {
		t.Errorf("LogFile = %q, want %q", cfg.LogFile, "/tmp/test.log")
	}
	if cfg.LogMaxBytes != 1048576 {
		t.Errorf("LogMaxBytes = %d, want %d", cfg.LogMaxBytes, 1048576)
	}
	if cfg.LogBackupCount != 5 {
		t.Errorf("LogBackupCount = %d, want 5", cfg.LogBackupCount)
	}
	if cfg.LogToStdout != false {
		t.Errorf("LogToStdout = %v, want false", cfg.LogToStdout)
	}
	if cfg.DynamicFPSEnabled != false {
		t.Errorf("DynamicFPSEnabled = %v, want false", cfg.DynamicFPSEnabled)
	}
	if cfg.CPULoadThreshold != 0.85 {
		t.Errorf("CPULoadThreshold = %f, want 0.85", cfg.CPULoadThreshold)
	}
	if cfg.CPUTempThresholdC != 80.0 {
		t.Errorf("CPUTempThresholdC = %f, want 80.0", cfg.CPUTempThresholdC)
	}
	if cfg.CaptureWidth != 1280 {
		t.Errorf("CaptureWidth = %d, want 1280", cfg.CaptureWidth)
	}
	if cfg.CaptureHeight != 720 {
		t.Errorf("CaptureHeight = %d, want 720", cfg.CaptureHeight)
	}
	if cfg.CaptureFPS != 30 {
		t.Errorf("CaptureFPS = %d, want 30", cfg.CaptureFPS)
	}
	if cfg.UIFPS != 25 {
		t.Errorf("UIFPS = %d, want 25", cfg.UIFPS)
	}
	if cfg.CameraSlotCount != 4 {
		t.Errorf("CameraSlotCount = %d, want 4", cfg.CameraSlotCount)
	}
	if cfg.KillDeviceHolders != false {
		t.Errorf("KillDeviceHolders = %v, want false", cfg.KillDeviceHolders)
	}
	if cfg.HealthLogIntervalSec != 60.0 {
		t.Errorf("HealthLogIntervalSec = %f, want 60.0", cfg.HealthLogIntervalSec)
	}
	if cfg.StaleFrameTimeoutSec != 2.0 {
		t.Errorf("StaleFrameTimeoutSec = %f, want 2.0", cfg.StaleFrameTimeoutSec)
	}
	if cfg.MaxRestartsPerWindow != 5 {
		t.Errorf("MaxRestartsPerWindow = %d, want 5", cfg.MaxRestartsPerWindow)
	}
}

func TestLoad_PartialINI(t *testing.T) {
	// Only override some values; rest should be defaults
	content := `
[profile]
capture_fps = 15
`
	tmp := writeTempFile(t, content)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.CaptureFPS != 15 {
		t.Errorf("CaptureFPS = %d, want 15", cfg.CaptureFPS)
	}
	// Defaults should be preserved
	def := DefaultConfig()
	if cfg.CaptureWidth != def.CaptureWidth {
		t.Errorf("CaptureWidth = %d, want default %d", cfg.CaptureWidth, def.CaptureWidth)
	}
	if cfg.UIFPS != def.UIFPS {
		t.Errorf("UIFPS = %d, want default %d", cfg.UIFPS, def.UIFPS)
	}
}

func TestLoad_EnvVarOverride(t *testing.T) {
	content := `
[logging]
file = /original/path.log
`
	tmp := writeTempFile(t, content)

	os.Setenv("CAMERA_DASHBOARD_LOG_FILE", "/env/override.log")
	defer os.Unsetenv("CAMERA_DASHBOARD_LOG_FILE")

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.LogFile != "/env/override.log" {
		t.Errorf("LogFile = %q, want %q", cfg.LogFile, "/env/override.log")
	}
}

func TestLoad_ClampedValues(t *testing.T) {
	content := `
[performance]
cpu_load_threshold = 0.01
cpu_temp_threshold_c = 200.0

[profile]
capture_width = 50
capture_height = 2000
capture_fps = 0
ui_fps = 100

[camera]
slot_count = 20
`
	tmp := writeTempFile(t, content)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	// cpu_load_threshold min is 0.1
	if cfg.CPULoadThreshold != 0.1 {
		t.Errorf("CPULoadThreshold = %f, want 0.1 (clamped)", cfg.CPULoadThreshold)
	}
	// cpu_temp_threshold_c max is 100.0
	if cfg.CPUTempThresholdC != 100.0 {
		t.Errorf("CPUTempThresholdC = %f, want 100.0 (clamped)", cfg.CPUTempThresholdC)
	}
	// capture_width min is 160
	if cfg.CaptureWidth != 160 {
		t.Errorf("CaptureWidth = %d, want 160 (clamped)", cfg.CaptureWidth)
	}
	// capture_height max is 1080
	if cfg.CaptureHeight != 1080 {
		t.Errorf("CaptureHeight = %d, want 1080 (clamped)", cfg.CaptureHeight)
	}
	// capture_fps min is 1
	if cfg.CaptureFPS != 1 {
		t.Errorf("CaptureFPS = %d, want 1 (clamped)", cfg.CaptureFPS)
	}
	// ui_fps max is 60
	if cfg.UIFPS != 60 {
		t.Errorf("UIFPS = %d, want 60 (clamped)", cfg.UIFPS)
	}
	// slot_count max is 8
	if cfg.CameraSlotCount != 8 {
		t.Errorf("CameraSlotCount = %d, want 8 (clamped)", cfg.CameraSlotCount)
	}
}

func TestLoad_CPULoadThresholdMaxClamp(t *testing.T) {
	content := `
[performance]
cpu_load_threshold = 5.0
`
	tmp := writeTempFile(t, content)

	cfg, err := Load(tmp)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.CPULoadThreshold != 1.0 {
		t.Errorf("CPULoadThreshold = %f, want 1.0 (clamped)", cfg.CPULoadThreshold)
	}
}

// =============================================================================
// ChooseProfile tests
// =============================================================================

func TestChooseProfile_SingleCamera(t *testing.T) {
	cfg := DefaultConfig()
	w, h, fps, uiFPS := cfg.ChooseProfile(1)

	if w != 640 || h != 480 {
		t.Errorf("1 camera: resolution = %dx%d, want 640x480", w, h)
	}
	if fps != 25 {
		t.Errorf("1 camera: captureFPS = %d, want 25", fps)
	}
	if uiFPS != 20 {
		t.Errorf("1 camera: uiFPS = %d, want 20", uiFPS)
	}
}

func TestChooseProfile_TwoCameras(t *testing.T) {
	cfg := DefaultConfig()
	w, h, fps, uiFPS := cfg.ChooseProfile(2)

	// Python parity: no pre-scaling by camera count
	if w != 640 || h != 480 {
		t.Errorf("2 cameras: resolution = %dx%d, want 640x480", w, h)
	}
	if fps != 25 {
		t.Errorf("2 cameras: captureFPS = %d, want 25", fps)
	}
	if uiFPS != 20 {
		t.Errorf("2 cameras: uiFPS = %d, want 20", uiFPS)
	}
}

func TestChooseProfile_FourCameras(t *testing.T) {
	cfg := DefaultConfig()
	w, h, fps, _ := cfg.ChooseProfile(4)

	if w != 640 {
		t.Errorf("4 cameras: width = %d, want 640", w)
	}
	if h != 480 {
		t.Errorf("4 cameras: height = %d, want 480", h)
	}
	if fps != 25 {
		t.Errorf("4 cameras: captureFPS = %d, want 25", fps)
	}
}

func TestChooseProfile_SixCameras(t *testing.T) {
	cfg := DefaultConfig()
	w, h, fps, _ := cfg.ChooseProfile(6)

	if w != 640 {
		t.Errorf("6 cameras: width = %d, want 640", w)
	}
	if h != 480 {
		t.Errorf("6 cameras: height = %d, want 480", h)
	}
	if fps != 25 {
		t.Errorf("6 cameras: captureFPS = %d, want 25", fps)
	}
}

func TestChooseProfile_NoImplicitRounding(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CaptureWidth = 500
	cfg.CaptureHeight = 300

	w, h, _, _ := cfg.ChooseProfile(1)

	if w != 500 {
		t.Errorf("width = %d, want 500", w)
	}
	if h != 300 {
		t.Errorf("height = %d, want 300", h)
	}
}

func TestChooseProfile_NoCameraCountScaling(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CaptureWidth = 320
	cfg.CaptureHeight = 240
	cfg.CaptureFPS = 17
	cfg.UIFPS = 13

	w, h, fps, uiFPS := cfg.ChooseProfile(6)

	if w != 320 || h != 240 {
		t.Errorf("6 cameras: resolution = %dx%d, want 320x240", w, h)
	}
	if fps != 17 || uiFPS != 13 {
		t.Errorf("6 cameras: fps/uiFPS = %d/%d, want 17/13", fps, uiFPS)
	}
}

// =============================================================================
// Validate tests
// =============================================================================

func TestValidate_DefaultsAreOK(t *testing.T) {
	cfg := DefaultConfig()
	ok, warnings := cfg.Validate()

	if !ok {
		t.Errorf("DefaultConfig().Validate() returned ok=false, warnings: %v", warnings)
	}
}

func TestValidate_HighResolutionWarning(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CaptureWidth = 1280
	cfg.CaptureHeight = 720 // 921600 pixels > 480000

	_, warnings := cfg.Validate()

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "High resolution") {
			found = true
		}
	}
	if !found {
		t.Error("expected high resolution warning")
	}
}

func TestValidate_HighFPSWarning(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CaptureFPS = 30

	_, warnings := cfg.Validate()

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "FPS 30") {
			found = true
		}
	}
	if !found {
		t.Error("expected high FPS warning")
	}
}

func TestValidate_BandwidthExceeded(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CaptureWidth = 1920
	cfg.CaptureHeight = 1080
	cfg.CaptureFPS = 30
	cfg.CameraSlotCount = 4

	ok, warnings := cfg.Validate()

	if ok {
		t.Error("expected Validate to return ok=false for excessive bandwidth")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "bandwidth") {
			found = true
		}
	}
	if !found {
		t.Error("expected bandwidth warning")
	}
}

func TestValidate_MinDynamicFPSTooHigh(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MinDynamicFPS = 30
	cfg.CaptureFPS = 15

	_, warnings := cfg.Validate()

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "MinDynamicFPS") {
			found = true
		}
	}
	if !found {
		t.Error("expected MinDynamicFPS warning")
	}
}

// =============================================================================
// roundDown16 tests
// =============================================================================

func TestRoundDown16(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{640, 640},
		{641, 640},
		{655, 640},
		{656, 656},
		{320, 320},
		{15, 0},
		{16, 16},
		{0, 0},
	}

	for _, tc := range tests {
		got := roundDown16(tc.input)
		if got != tc.want {
			t.Errorf("roundDown16(%d) = %d, want %d", tc.input, got, tc.want)
		}
	}
}

// =============================================================================
// ConfigPath tests
// =============================================================================

func TestConfigPath_Default(t *testing.T) {
	os.Unsetenv("CAMERA_DASHBOARD_CONFIG")
	path := ConfigPath()
	if path != "./config.ini" {
		t.Errorf("ConfigPath() = %q, want %q", path, "./config.ini")
	}
}

func TestConfigPath_EnvOverride(t *testing.T) {
	os.Setenv("CAMERA_DASHBOARD_CONFIG", "/custom/path.ini")
	defer os.Unsetenv("CAMERA_DASHBOARD_CONFIG")

	path := ConfigPath()
	if path != "/custom/path.ini" {
		t.Errorf("ConfigPath() = %q, want %q", path, "/custom/path.ini")
	}
}

// =============================================================================
// Helper
// =============================================================================

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "test.ini")
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return tmp
}
