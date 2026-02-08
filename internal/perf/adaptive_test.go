package perf

import (
	"camera-dashboard-go/internal/config"
	"testing"
)

func TestNewSmartController_NilConfig(t *testing.T) {
	sc := NewSmartController(nil, nil)

	if sc.dynamicEnabled {
		t.Error("nil config should disable dynamic FPS")
	}
	if sc.GetCurrentFPS() <= 0 {
		t.Errorf("current FPS = %d, should be > 0", sc.GetCurrentFPS())
	}
	if sc.GetState() != "" && sc.GetState() != "Probing" && sc.GetState() != "Stable" {
		// before Start(), state is 0 = Probing
	}
}

func TestNewSmartController_DynamicEnabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = true
	cfg.CaptureFPS = 20
	cfg.MinDynamicFPS = 10

	sc := NewSmartController(nil, cfg)

	if !sc.dynamicEnabled {
		t.Error("dynamic FPS should be enabled")
	}
	if sc.GetCurrentFPS() != 20 {
		t.Errorf("current FPS = %d, want 20", sc.GetCurrentFPS())
	}
	if sc.minFPS != 10 {
		t.Errorf("minFPS = %d, want 10", sc.minFPS)
	}
	if sc.maxFPS != 20 {
		t.Errorf("maxFPS = %d, want 20", sc.maxFPS)
	}
}

func TestNewSmartController_DynamicDisabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = false
	cfg.CaptureFPS = 15

	sc := NewSmartController(nil, cfg)

	if sc.dynamicEnabled {
		t.Error("dynamic FPS should be disabled")
	}
	// In fixed mode, min == max == captureFPS
	if sc.minFPS != 15 {
		t.Errorf("fixed mode minFPS = %d, want 15", sc.minFPS)
	}
	if sc.maxFPS != 15 {
		t.Errorf("fixed mode maxFPS = %d, want 15", sc.maxFPS)
	}
	if sc.GetCurrentFPS() != 15 {
		t.Errorf("current FPS = %d, want 15", sc.GetCurrentFPS())
	}
}

func TestNewSmartController_MinFPSClamped(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = true
	cfg.CaptureFPS = 20
	cfg.MinDynamicFPS = 3 // below MinFPS constant (10)

	sc := NewSmartController(nil, cfg)

	if sc.minFPS != MinFPS {
		t.Errorf("minFPS = %d, want %d (clamped to MinFPS)", sc.minFPS, MinFPS)
	}
}

func TestNewSmartController_CaptureFPSClampedToMax(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = true
	cfg.CaptureFPS = 50 // above MaxFPS (30)
	cfg.MinDynamicFPS = 10

	sc := NewSmartController(nil, cfg)

	if sc.maxFPS != MaxFPS {
		t.Errorf("maxFPS = %d, want %d (clamped)", sc.maxFPS, MaxFPS)
	}
	if sc.GetCurrentFPS() != MaxFPS {
		t.Errorf("currentFPS = %d, want %d", sc.GetCurrentFPS(), MaxFPS)
	}
}

func TestChangeFPS_Clamping(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = true
	cfg.CaptureFPS = 20
	cfg.MinDynamicFPS = 10

	sc := NewSmartController(nil, cfg)

	// clamp below min
	sc.changeFPS(5)
	if sc.GetCurrentFPS() != 10 {
		t.Errorf("changeFPS(5): FPS = %d, want 10 (clamped to min)", sc.GetCurrentFPS())
	}

	// clamp above max
	sc.changeFPS(30)
	if sc.GetCurrentFPS() != 20 {
		t.Errorf("changeFPS(30): FPS = %d, want 20 (clamped to max)", sc.GetCurrentFPS())
	}

	// within range
	sc.changeFPS(15)
	if sc.GetCurrentFPS() != 15 {
		t.Errorf("changeFPS(15): FPS = %d, want 15", sc.GetCurrentFPS())
	}
}

func TestChangeFPS_NoOpWhenSame(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = true
	cfg.CaptureFPS = 15
	cfg.MinDynamicFPS = 10

	sc := NewSmartController(nil, cfg)
	initialAdjustCount := sc.adjustCount

	sc.changeFPS(15) // same as current
	if sc.adjustCount != initialAdjustCount {
		t.Error("changeFPS should be no-op when FPS unchanged")
	}
}

func TestGetState_Names(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = true
	cfg.CaptureFPS = 15
	cfg.MinDynamicFPS = 10

	sc := NewSmartController(nil, cfg)

	sc.state.Store(StateProbing)
	if sc.GetState() != "Probing" {
		t.Errorf("GetState() = %q, want Probing", sc.GetState())
	}

	sc.state.Store(StateStable)
	if sc.GetState() != "Stable" {
		t.Errorf("GetState() = %q, want Stable", sc.GetState())
	}

	sc.state.Store(StateRecovering)
	if sc.GetState() != "Recovering" {
		t.Errorf("GetState() = %q, want Recovering", sc.GetState())
	}

	sc.state.Store(StateEmergency)
	if sc.GetState() != "Emergency" {
		t.Errorf("GetState() = %q, want Emergency", sc.GetState())
	}
}

func TestIsDynamic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = true
	sc := NewSmartController(nil, cfg)
	if !sc.IsDynamic() {
		t.Error("IsDynamic() should return true when dynamic is enabled")
	}

	cfg.DynamicFPSEnabled = false
	sc = NewSmartController(nil, cfg)
	if sc.IsDynamic() {
		t.Error("IsDynamic() should return false when dynamic is disabled")
	}
}

func TestGetSweetSpotFPS(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.DynamicFPSEnabled = true
	cfg.CaptureFPS = 20
	cfg.MinDynamicFPS = 10

	sc := NewSmartController(nil, cfg)
	if sc.GetSweetSpotFPS() != 20 {
		t.Errorf("initial sweet spot = %d, want 20 (same as captureFPS)", sc.GetSweetSpotFPS())
	}
}

func TestNewAdaptiveController_Alias(t *testing.T) {
	// NewAdaptiveController is an alias for NewSmartController
	cfg := config.DefaultConfig()
	cfg.CaptureFPS = 15
	ac := NewAdaptiveController(nil, cfg)
	if ac.GetCurrentFPS() != 15 {
		t.Errorf("NewAdaptiveController FPS = %d, want 15", ac.GetCurrentFPS())
	}
}
