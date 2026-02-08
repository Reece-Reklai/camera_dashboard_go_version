# Camera Dashboard

Opencode Driven: A high-performance 3-camera setup as a monitoring system for Raspberry Pi. Built with Go and the Fyne GUI framework.

## Features

- **Multi-Camera Support** - Up to 3 USB cameras in a dynamic smart grid layout
- **Real-time Video** - Configurable resolution/FPS (default 640x480 @ 25 FPS), optimized for vehicle monitoring
- **Touch Interface** - Tap for fullscreen, long-press to swap camera positions
- **Hot-plug Detection** - Sysfs-based USB parent matching to avoid false positives from multi-function cameras; per-camera restart on disconnect/reconnect (other cameras unaffected)
- **Adaptive FPS** - Dynamic thermal/load-based FPS scaling with emergency throttle and sweet-spot probing
- **Night Mode** - LUT-based red-channel night vision filter (toggle via UI)
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
| **Exit button** | Clean shutdown |

## Configuration

Edit `config.ini` (or set environment variables) to change settings:

```ini
[capture]
width = 640       # Resolution width (160-1920)
height = 480      # Resolution height (120-1080)
fps = 25          # Frames per second (1-60)

[performance]
dynamic_fps_enabled = true
min_dynamic_fps = 5

[recovery]
stale_frame_timeout_sec = 10
restart_cooldown_sec = 30
```

Set `CAM_DASH_CONFIG` to override config path. Then rebuild: `make build`

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

The hotplug scanner polls `/dev/video*` every 2 seconds using sysfs (not `v4l2-ctl`) to avoid conflicts with active FFmpeg captures. Multi-function USB cameras register multiple `/dev/videoX` nodes under the same physical USB device (e.g., a UVC webcam may own video0-video3). To prevent false "new camera" detections, the scanner resolves each candidate's sysfs USB parent path and rejects any device that shares a parent with an already-tracked camera.

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
