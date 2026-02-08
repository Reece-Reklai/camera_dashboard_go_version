package camera

// =============================================================================
// Camera Settings
// =============================================================================
// Settings are loaded from config.ini via the config package.
// Default values here are used as fallbacks only.
// =============================================================================

// Default values (used when no config is provided)
const (
	DefaultWidth  = 640
	DefaultHeight = 480
	DefaultFPS    = 15
	DefaultFormat = "mjpeg"
)

// Settings holds camera capture configuration.
// Populated from config.Config at startup and passed to the Manager.
type Settings struct {
	Width  int    // Capture width in pixels
	Height int    // Capture height in pixels
	FPS    int    // Target frames per second
	Format string // Capture format: "mjpeg" or "yuyv"
}

// DefaultSettings returns sensible defaults for vehicle camera monitoring.
func DefaultSettings() Settings {
	return Settings{
		Width:  DefaultWidth,
		Height: DefaultHeight,
		FPS:    DefaultFPS,
		Format: DefaultFormat,
	}
}
