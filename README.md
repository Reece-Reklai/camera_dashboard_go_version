# Camera Dashboard

Opencode Driven: A high-performance multi-camera blindspot monitoring system for Raspberry Pi. Built with Go and the Fyne GUI framework.

## Features

- **Multi-Camera Support** - Configurable camera slots (`slot_count`, default 3, max 8) in a dynamic smart grid layout
- **Real-time Video** - Configurable resolution/FPS (default 640x480 @ 25 FPS), optimized for vehicle monitoring
- **Touch Interface** - Tap for fullscreen, long-press to swap camera positions
- **Hot-plug Detection** - Sysfs-based USB parent matching to avoid false positives from multi-function cameras; per-camera restart on disconnect/reconnect (other cameras unaffected)
- **Adaptive FPS** - Dynamic thermal/load-based FPS scaling with emergency throttle and sweet-spot probing
- **Night Mode** - LUT-based red-channel night vision filter (toggle via UI)
- **Brightness Presets** - Settings tile supports 15%, 60%, 80%, 100%, 150% brightness levels
- **Clean Shutdown** - Capture workers check stop signals before FFmpeg format fallback retries, preventing zombie processes during exit
- **Low Power** - Optimized for battery-powered operation (~100% CPU for 2 cameras)
- **Single Binary** - No Python, no runtime dependencies

## Quick Start

### On a Raspberry Pi (64-bit OS)

```bash
# Download and extract
tar -xzvf camera-dashboard-*.tar.gz

# Install (installs dependencies + binary)
./install.sh

# Run
DISPLAY=:0 camera-dashboard
```

### Build from Source

```bash
# Install dependencies
sudo apt install ffmpeg v4l-utils

# Build
make build

# Run
make run
```

## Requirements

| Requirement | Details |
|-------------|---------|
| **OS** | Raspberry Pi OS (64-bit) or Ubuntu ARM64 |
| **Hardware** | Raspberry Pi 3/4/5 |
| **Display** | X11 desktop environment |
| **Cameras** | USB cameras with V4L2 support |
| **Dependencies** | ffmpeg, v4l-utils |

## Usage

| Action | Result |
|--------|--------|
| **Tap camera** | Fullscreen view |
| **Tap fullscreen** | Exit fullscreen |
| **Long-press camera** | Enter swap mode |
| **Tap another slot** | Swap positions |
| **Restart button** | Reinitialize cameras |
| **Brightness buttons** | Adjust display brightness (15/60/80/100/150%) |
| **Exit button** | Clean shutdown |

## Configuration

Edit `config.ini` (or set environment variables) to change settings:

```ini
[profile]
capture_width = 640
capture_height = 480
capture_fps = 25
capture_format = mjpeg
ui_fps = 20

[performance]
dynamic_fps = true
cpu_load_threshold = 0.75
cpu_temp_threshold_c = 75.0
min_dynamic_fps = 10
stale_frame_timeout_sec = 1.5
restart_cooldown_sec = 5.0

[camera]
slot_count = 3
kill_device_holders = true
```

Set `CAMERA_DASHBOARD_CONFIG` to override config path. Then rebuild: `make build`

## Makefile Targets

```bash
make build      # Development build
make release    # Optimized build
make package    # Create deployment tarball
make run        # Build and run
make status     # Show CPU/temp/memory
make clean      # Remove build artifacts
make help       # Show all targets
```

## Project Structure

```
.
├── main.go                 # Entry point, signal handling
├── config.ini              # Runtime configuration (optional)
├── internal/
│   ├── camera/
│   │   ├── config.go       # Camera Settings struct + defaults
│   │   ├── manager.go      # Camera lifecycle management
│   │   ├── capture.go      # FFmpeg capture, frame decoding, clean shutdown
│   │   ├── framebuffer.go  # Thread-safe double-buffered frame storage
│   │   └── device.go       # Camera discovery (v4l2, sysfs)
│   ├── config/
│   │   ├── config.go       # INI loading, profiles, validation
│   │   └── logging.go      # Rotating file writer
│   ├── helpers/
│   │   ├── grid.go             # Smart grid layout calculator
│   │   └── kill_device_holders.go  # Stale process cleanup
│   ├── ui/
│   │   ├── app.go          # Fyne application, full UI, hotplug (sysfs USB parent matching)
│   │   └── nightmode.go    # Night mode LUT + filter
│   └── perf/
│       ├── adaptive.go     # Adaptive FPS controller
│       └── monitor.go      # CPU/temperature monitoring
├── Makefile                # Build system
├── install.sh              # Deployment installer
```

## Architecture Notes

### Hot-plug Detection

The hotplug scanner polls `/dev/video*` on a config-driven interval (`[camera] rescan_interval_ms`, default `15000`) using sysfs (not `v4l2-ctl`) to avoid conflicts with active FFmpeg captures. Multi-function USB cameras register multiple `/dev/videoX` nodes under the same physical USB device (e.g., a UVC webcam may own video0-video3). To prevent false "new camera" detections, the scanner resolves each candidate's sysfs USB parent path and rejects any device that shares a parent with an already-tracked camera.

### Capture & Shutdown

Each capture worker runs FFmpeg with format fallbacks (mjpeg -> yuyv422 -> auto). The format retry loop checks `cw.running` before each attempt, ensuring that when `Stop()` is called and FFmpeg is killed, the worker exits immediately rather than spawning a new FFmpeg process with the next format.

### Frame Buffer

Double-buffered with `sync.RWMutex` protecting `frames[]` access. Atomic indices coordinate writer (capture goroutine) and readers (UI goroutine). The mutex prevents data races on the `image.Image` interface values stored in the buffer slots.

## Troubleshooting

### No cameras detected
```bash
v4l2-ctl --list-devices  # Check if cameras are visible
ls -la /dev/video*       # Check device files
sudo usermod -a -G video $USER  # Add user to video group
```

### High CPU usage
- Reduce FPS in `config.ini` (try `fps = 10`)
- Use MJPEG format (not YUYV)
- Check for zombie processes: `ps aux | awk '$8 == "Z"'`

### Display issues
```bash
echo $DISPLAY  # Should be :0
DISPLAY=:0 camera-dashboard  # Set explicitly
```

## License

MIT License - see LICENSE.MIT
