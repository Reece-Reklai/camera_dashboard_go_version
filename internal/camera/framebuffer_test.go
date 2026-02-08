package camera

import (
	"image"
	"image/color"
	"sync"
	"testing"
	"time"
)

func makeTestImage(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestNewFrameBuffer_InitialState(t *testing.T) {
	fb := NewFrameBuffer()

	if fb.GetFrameCount() != 0 {
		t.Errorf("initial frame count = %d, want 0", fb.GetFrameCount())
	}

	if fb.GetDroppedCount() != 0 {
		t.Errorf("initial dropped count = %d, want 0", fb.GetDroppedCount())
	}

	lastTime := fb.GetLastFrameTime()
	if !lastTime.IsZero() {
		t.Errorf("initial last frame time should be zero, got %v", lastTime)
	}

	// Read before any write should return nil
	frame := fb.Read()
	if frame != nil {
		t.Error("Read() before any Write() should return nil")
	}
}

func TestFrameBuffer_WriteAndRead(t *testing.T) {
	fb := NewFrameBuffer()
	img := makeTestImage(10, 10, color.White)

	fb.Write(img)

	frame := fb.Read()
	if frame == nil {
		t.Fatal("Read() returned nil after Write()")
	}

	if fb.GetFrameCount() != 1 {
		t.Errorf("frame count = %d, want 1", fb.GetFrameCount())
	}

	lastTime := fb.GetLastFrameTime()
	if lastTime.IsZero() {
		t.Error("last frame time should not be zero after Write()")
	}
	if time.Since(lastTime) > time.Second {
		t.Error("last frame time is too old")
	}
}

func TestFrameBuffer_ReadIfNew(t *testing.T) {
	fb := NewFrameBuffer()

	// No frames yet
	frame, lastRead, hasNew := fb.ReadIfNew(0)
	if hasNew {
		t.Error("ReadIfNew(0) should return hasNew=false when no frames written")
	}
	if frame != nil {
		t.Error("ReadIfNew should return nil frame when no new frame")
	}

	// Write a frame
	img1 := makeTestImage(10, 10, color.White)
	fb.Write(img1)

	frame, lastRead, hasNew = fb.ReadIfNew(0)
	if !hasNew {
		t.Error("ReadIfNew(0) should return hasNew=true after Write()")
	}
	if frame == nil {
		t.Error("ReadIfNew should return non-nil frame when new")
	}
	if lastRead != 1 {
		t.Errorf("lastRead = %d, want 1", lastRead)
	}

	// Read again with same lastRead — should report no new frame
	frame, _, hasNew = fb.ReadIfNew(lastRead)
	if hasNew {
		t.Error("ReadIfNew(1) should return hasNew=false (no new frame)")
	}

	// Write another frame
	img2 := makeTestImage(10, 10, color.Black)
	fb.Write(img2)

	frame, lastRead, hasNew = fb.ReadIfNew(1)
	if !hasNew {
		t.Error("ReadIfNew(1) should return hasNew=true after second Write()")
	}
	if lastRead != 2 {
		t.Errorf("lastRead = %d, want 2", lastRead)
	}
}

func TestFrameBuffer_GetFrameCount(t *testing.T) {
	fb := NewFrameBuffer()

	for i := 0; i < 10; i++ {
		fb.Write(makeTestImage(1, 1, color.White))
	}

	if fb.GetFrameCount() != 10 {
		t.Errorf("frame count = %d, want 10", fb.GetFrameCount())
	}
}

func TestFrameBuffer_Reset(t *testing.T) {
	fb := NewFrameBuffer()
	fb.Write(makeTestImage(10, 10, color.White))
	fb.MarkDropped()
	fb.MarkDropped()

	fb.Reset()

	if fb.GetFrameCount() != 0 {
		t.Errorf("frame count after reset = %d, want 0", fb.GetFrameCount())
	}
	if fb.GetDroppedCount() != 0 {
		t.Errorf("dropped count after reset = %d, want 0", fb.GetDroppedCount())
	}
	if !fb.GetLastFrameTime().IsZero() {
		t.Error("last frame time should be zero after reset")
	}
}

func TestFrameBuffer_MarkDropped(t *testing.T) {
	fb := NewFrameBuffer()
	fb.MarkDropped()
	fb.MarkDropped()
	fb.MarkDropped()

	if fb.GetDroppedCount() != 3 {
		t.Errorf("dropped count = %d, want 3", fb.GetDroppedCount())
	}
}

func TestFrameBuffer_GetCaptureStats(t *testing.T) {
	fb := NewFrameBuffer()
	for i := 0; i < 100; i++ {
		fb.Write(makeTestImage(1, 1, color.White))
	}

	fps, total, uptime := fb.GetCaptureStats()
	if total != 100 {
		t.Errorf("total frames = %d, want 100", total)
	}
	if uptime <= 0 {
		t.Error("uptime should be positive")
	}
	if fps <= 0 {
		t.Error("fps should be positive")
	}
}

func TestFrameBuffer_ConcurrentSafety(t *testing.T) {
	fb := NewFrameBuffer()
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			fb.Write(makeTestImage(1, 1, color.White))
		}
	}()

	// Reader goroutines
	for r := 0; r < 3; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var lastRead uint64
			for i := 0; i < 1000; i++ {
				_, lr, _ := fb.ReadIfNew(lastRead)
				lastRead = lr
				fb.Read()
				fb.GetFrameCount()
				fb.GetDroppedCount()
			}
		}()
	}

	wg.Wait()

	if fb.GetFrameCount() != 1000 {
		t.Errorf("frame count after concurrent test = %d, want 1000", fb.GetFrameCount())
	}
}
