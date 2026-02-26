package perf

import "testing"

func TestNormalizeLoadAverage(t *testing.T) {
	tests := []struct {
		name     string
		load1    float64
		cpuCount int
		want     float64
	}{
		{name: "half load", load1: 2.0, cpuCount: 4, want: 0.5},
		{name: "clamped high", load1: 10.0, cpuCount: 4, want: 1.0},
		{name: "negative clamped", load1: -1.0, cpuCount: 4, want: 0.0},
		{name: "zero cpu fallback", load1: 1.0, cpuCount: 0, want: 1.0},
	}

	for _, tc := range tests {
		got := normalizeLoadAverage(tc.load1, tc.cpuCount)
		if got != tc.want {
			t.Errorf("%s: normalizeLoadAverage(%v, %d) = %v, want %v",
				tc.name, tc.load1, tc.cpuCount, got, tc.want)
		}
	}
}

func TestMonitorIsUnderStressThresholds(t *testing.T) {
	m := NewMonitor()

	// Below both thresholds
	m.loadAvg = 0.5
	m.temperature = 60.0
	if m.IsUnderStress() {
		t.Fatal("expected not under stress")
	}

	// Load threshold
	m.loadAvg = 0.75
	m.temperature = 60.0
	if !m.IsUnderStress() {
		t.Fatal("expected under stress by load")
	}

	// Temp threshold
	m.loadAvg = 0.2
	m.temperature = 75.0
	if !m.IsUnderStress() {
		t.Fatal("expected under stress by temperature")
	}
}
