package camera

import (
	"fmt"
	"image"
	"log"
	"sync"
	"time"
)

// Manager manages multiple cameras and capture workers
type Manager struct {
	cameras       []Camera
	workers       []*CaptureWorker
	frameChannels map[string]chan image.Image // Legacy channel mode
	frameBuffers  map[string]*FrameBuffer     // New buffer mode (preferred)
	useBufferMode bool                        // If true, use FrameBuffer instead of channels
	settings      Settings                    // Camera capture settings from config
	running       bool
	mutex         sync.RWMutex
}

// NewManager creates a new camera manager (legacy channel mode)
func NewManager() *Manager {
	return &Manager{
		frameChannels: make(map[string]chan image.Image),
		frameBuffers:  make(map[string]*FrameBuffer),
		useBufferMode: false,
		settings:      DefaultSettings(),
	}
}

// NewManagerWithBuffers creates a manager using FrameBuffer mode (recommended)
func NewManagerWithBuffers() *Manager {
	return &Manager{
		frameChannels: make(map[string]chan image.Image),
		frameBuffers:  make(map[string]*FrameBuffer),
		useBufferMode: true,
		settings:      DefaultSettings(),
	}
}

// NewManagerWithSettings creates a manager with explicit settings from config
func NewManagerWithSettings(s Settings, useBuffers bool) *Manager {
	// Apply defaults for zero values
	if s.Width == 0 {
		s.Width = DefaultWidth
	}
	if s.Height == 0 {
		s.Height = DefaultHeight
	}
	if s.FPS == 0 {
		s.FPS = DefaultFPS
	}
	if s.Format == "" {
		s.Format = DefaultFormat
	}

	return &Manager{
		frameChannels: make(map[string]chan image.Image),
		frameBuffers:  make(map[string]*FrameBuffer),
		useBufferMode: useBuffers,
		settings:      s,
	}
}

// GetSettings returns the manager's camera settings
func (m *Manager) GetSettings() Settings {
	return m.settings
}

// Initialize discovers and initializes cameras
func (m *Manager) Initialize() error {
	log.Println("[Manager] Stopping existing workers...")
	// Stop existing workers (without holding mutex - stopInternal handles its own locking)
	m.stopInternal()

	m.mutex.Lock()
	defer m.mutex.Unlock()

	log.Println("[Manager] Discovering cameras...")
	// Discover cameras
	cameras, err := DiscoverCamerasWithSettings(m.settings)
	if err != nil {
		log.Printf("[Manager] Camera discovery failed: %v", err)
		return err
	}

	log.Printf("[Manager] Found %d cameras", len(cameras))
	m.cameras = cameras
	m.workers = make([]*CaptureWorker, len(cameras))
	m.frameChannels = make(map[string]chan image.Image)
	m.frameBuffers = make(map[string]*FrameBuffer)

	// Create capture workers for each camera
	for i, camera := range cameras {
		log.Printf("[Manager] Creating worker for camera %s (%s) [buffer mode: %v]",
			camera.DeviceID, camera.DevicePath, m.useBufferMode)

		var worker *CaptureWorker

		if m.useBufferMode {
			// New FrameBuffer mode - decoupled capture from UI
			buffer := NewFrameBuffer()
			worker = NewCaptureWorkerWithBuffer(camera, buffer, m.settings)
			m.frameBuffers[camera.DeviceID] = buffer
		} else {
			// Legacy channel mode
			frameCh := make(chan image.Image, 1) // Latest-frame-only buffer
			worker = NewCaptureWorker(camera, frameCh, m.settings)
			m.frameChannels[camera.DeviceID] = frameCh
		}

		m.workers[i] = worker
	}

	m.running = true
	log.Println("[Manager] Initialization complete")
	return nil
}

// Start starts all camera capture workers with staggered timing
// to reduce USB bandwidth contention during initialization
func (m *Manager) Start() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.running {
		return ErrManagerNotInitialized
	}

	// Start cameras with staggered delays to reduce USB bandwidth contention
	// USB 2.0 bandwidth is limited (~35MB/s real-world), starting all cameras
	// simultaneously causes buffer overruns on some cameras
	for i, worker := range m.workers {
		if i > 0 {
			// Wait 500ms between camera starts to allow USB subsystem to stabilize
			log.Printf("[Manager] Waiting 500ms before starting camera %d to reduce USB contention", i+1)
			time.Sleep(500 * time.Millisecond)
		}

		if err := worker.Start(); err != nil {
			return err
		}
		log.Printf("[Manager] Started camera %d/%d", i+1, len(m.workers))
	}

	return nil
}

// Stop stops all camera capture workers
func (m *Manager) Stop() {
	m.stopInternal()
}

// stopInternal stops all workers (with its own locking)
func (m *Manager) stopInternal() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.running = false

	// Stop all workers
	for _, worker := range m.workers {
		if worker != nil {
			worker.Stop()
		}
	}

	// Close all frame channels (legacy mode)
	for _, ch := range m.frameChannels {
		close(ch)
	}

	m.workers = nil
	m.frameChannels = make(map[string]chan image.Image)
	m.frameBuffers = make(map[string]*FrameBuffer)
}

// GetCameras returns list of cameras
func (m *Manager) GetCameras() []Camera {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	cameras := make([]Camera, len(m.cameras))
	copy(cameras, m.cameras)
	return cameras
}

// GetFrameChannel returns frame channel for a specific camera
func (m *Manager) GetFrameChannel(cameraID string) <-chan image.Image {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if ch, exists := m.frameChannels[cameraID]; exists {
		return ch
	}

	return nil
}

// GetFrameBuffer returns frame buffer for a specific camera (new mode)
func (m *Manager) GetFrameBuffer(cameraID string) *FrameBuffer {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if buf, exists := m.frameBuffers[cameraID]; exists {
		return buf
	}

	return nil
}

// IsBufferMode returns true if manager is using FrameBuffer mode
func (m *Manager) IsBufferMode() bool {
	return m.useBufferMode
}

// SetFPS sets the FPS for all capture workers
func (m *Manager) SetFPS(fps int) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, worker := range m.workers {
		if worker != nil {
			worker.SetFPS(fps)
		}
	}
}

// GetWorker returns the capture worker for a specific camera
func (m *Manager) GetWorker(cameraID string) *CaptureWorker {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for i, cam := range m.cameras {
		if cam.DeviceID == cameraID && i < len(m.workers) {
			return m.workers[i]
		}
	}
	return nil
}

// RestartCamera restarts only the specified camera's capture worker
// Other cameras continue running uninterrupted
func (m *Manager) RestartCamera(cameraID string) error {
	m.mutex.RLock()
	var worker *CaptureWorker
	for i, cam := range m.cameras {
		if cam.DeviceID == cameraID && i < len(m.workers) {
			worker = m.workers[i]
			break
		}
	}
	m.mutex.RUnlock()

	if worker == nil {
		return fmt.Errorf("camera %s not found", cameraID)
	}

	log.Printf("[Manager] Restarting camera %s (other cameras unaffected)", cameraID)
	return worker.Restart()
}

// RestartCameraByIndex restarts only the camera at the specified index
// Other cameras continue running uninterrupted
func (m *Manager) RestartCameraByIndex(index int) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if index < 0 || index >= len(m.workers) {
		return fmt.Errorf("camera index %d out of range", index)
	}

	worker := m.workers[index]
	if worker == nil {
		return fmt.Errorf("camera at index %d has no worker", index)
	}

	log.Printf("[Manager] Restarting camera at index %d (other cameras unaffected)", index)
	return worker.Restart()
}

// Errors
var (
	ErrManagerNotInitialized = fmt.Errorf("camera manager not initialized")
)
