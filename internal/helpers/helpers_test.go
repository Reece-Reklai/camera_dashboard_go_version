package helpers

import (
	"os"
	"syscall"
	"testing"
)

// ===========================================================================
// GetSmartGrid tests
// ===========================================================================

func TestGetSmartGrid_ZeroOrOne(t *testing.T) {
	for _, n := range []int{0, 1} {
		rows, cols := GetSmartGrid(n)
		if rows != 1 || cols != 1 {
			t.Errorf("GetSmartGrid(%d) = (%d,%d), want (1,1)", n, rows, cols)
		}
	}
}

func TestGetSmartGrid_Two(t *testing.T) {
	rows, cols := GetSmartGrid(2)
	if rows != 1 || cols != 2 {
		t.Errorf("GetSmartGrid(2) = (%d,%d), want (1,2)", rows, cols)
	}
}

func TestGetSmartGrid_Three(t *testing.T) {
	rows, cols := GetSmartGrid(3)
	if rows != 1 || cols != 3 {
		t.Errorf("GetSmartGrid(3) = (%d,%d), want (1,3)", rows, cols)
	}
}

func TestGetSmartGrid_Four(t *testing.T) {
	rows, cols := GetSmartGrid(4)
	if rows != 2 || cols != 2 {
		t.Errorf("GetSmartGrid(4) = (%d,%d), want (2,2)", rows, cols)
	}
}

func TestGetSmartGrid_FiveAndSix(t *testing.T) {
	for _, n := range []int{5, 6} {
		rows, cols := GetSmartGrid(n)
		if rows != 2 || cols != 3 {
			t.Errorf("GetSmartGrid(%d) = (%d,%d), want (2,3)", n, rows, cols)
		}
	}
}

func TestGetSmartGrid_SevenToNine(t *testing.T) {
	for _, n := range []int{7, 8, 9} {
		rows, cols := GetSmartGrid(n)
		if rows != 3 || cols != 3 {
			t.Errorf("GetSmartGrid(%d) = (%d,%d), want (3,3)", n, rows, cols)
		}
	}
}

func TestGetSmartGrid_TenPlus(t *testing.T) {
	// For 10+, cols should be <= 4 and rows*cols >= n
	for _, n := range []int{10, 12, 16, 25} {
		rows, cols := GetSmartGrid(n)
		if cols > 4 {
			t.Errorf("GetSmartGrid(%d): cols=%d > 4", n, cols)
		}
		if cols < 1 {
			t.Errorf("GetSmartGrid(%d): cols=%d < 1", n, cols)
		}
		if rows*cols < n {
			t.Errorf("GetSmartGrid(%d) = (%d,%d): product %d < n", n, rows, cols, rows*cols)
		}
	}
}

// ===========================================================================
// isqrt tests
// ===========================================================================

func TestIsqrt(t *testing.T) {
	tests := []struct {
		input int
		want  int
	}{
		{0, 0},
		{1, 1},
		{4, 2},
		{9, 3},
		{10, 3},
		{15, 3},
		{16, 4},
		{25, 5},
		{100, 10},
	}

	for _, tt := range tests {
		got := isqrt(tt.input)
		if got != tt.want {
			t.Errorf("isqrt(%d) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestIsqrt_Negative(t *testing.T) {
	got := isqrt(-5)
	if got != 0 {
		t.Errorf("isqrt(-5) = %d, want 0", got)
	}
}

// ===========================================================================
// KillDeviceHolders tests
// ===========================================================================

func TestKillDeviceHolders_DisabledIsNoOp(t *testing.T) {
	result := KillDeviceHolders("/dev/video99", false)
	if result {
		t.Error("KillDeviceHolders with enabled=false should return false")
	}
}

// ===========================================================================
// sortedKeys tests
// ===========================================================================

func TestSortedKeys(t *testing.T) {
	m := map[int]struct{}{
		5: {}, 1: {}, 3: {}, 2: {}, 4: {},
	}
	got := sortedKeys(m)
	want := []int{1, 2, 3, 4, 5}
	if len(got) != len(want) {
		t.Fatalf("sortedKeys length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("sortedKeys[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestSortedKeys_Empty(t *testing.T) {
	got := sortedKeys(map[int]struct{}{})
	if len(got) != 0 {
		t.Errorf("sortedKeys(empty) length = %d, want 0", len(got))
	}
}

func TestSortedKeys_Single(t *testing.T) {
	got := sortedKeys(map[int]struct{}{42: {}})
	if len(got) != 1 || got[0] != 42 {
		t.Errorf("sortedKeys(single) = %v, want [42]", got)
	}
}

// ===========================================================================
// isPermissionError tests
// ===========================================================================

func TestIsPermissionError(t *testing.T) {
	if !isPermissionError(syscall.EPERM) {
		t.Error("isPermissionError(EPERM) should be true")
	}
	if !isPermissionError(syscall.EACCES) {
		t.Error("isPermissionError(EACCES) should be true")
	}
	if isPermissionError(syscall.ENOENT) {
		t.Error("isPermissionError(ENOENT) should be false")
	}
	if isPermissionError(nil) {
		t.Error("isPermissionError(nil) should be false")
	}
}

// ===========================================================================
// isPIDAlive tests
// ===========================================================================

func TestIsPIDAlive_Self(t *testing.T) {
	// Our own PID should be alive
	if !isPIDAlive(os.Getpid()) {
		t.Error("isPIDAlive(self) should be true")
	}
}

func TestIsPIDAlive_Nonexistent(t *testing.T) {
	// PID 2147483647 (max int32) is extremely unlikely to exist
	if isPIDAlive(2147483647) {
		t.Skip("PID 2147483647 actually exists (unlikely)")
	}
}
