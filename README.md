# Frame Extraction Worker

Extracts JPEG frames from video files using ffmpeg and publishes them to Redis.

## Architecture

```
Video File → ffmpeg (-re) → Go Worker → Redis
```

- **ffmpeg**: Reads video file in real-time, outputs JPEG frames to stdout
- **Go worker**: Parses JPEG boundaries (SOI/EOI markers), publishes latest frame to Redis
- **Redis**: Temporary cache with TTL, one key per stream

## Requirements

- Go 1.21+
- ffmpeg
- Redis 7

## Quick Start

1. Copy environment file:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` with your video path:
   ```bash
   VIDEO_PATH=/path/to/your/video.mp4
   STREAM_ID=my-stream
   ```

3. Start Redis:
   ```bash
   docker run -d --name redis -p 6379:6379 redis:7
   ```

4. Run the worker:
   ```bash
   go run ./cmd/worker
   ```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `VIDEO_PATH` | (required) | Path to video file |
| `STREAM_ID` | `default` | Unique stream identifier |
| `REDIS_ADDR` | `localhost:6379` | Redis connection string |
| `FRAME_RATE` | `1` | Frames per second (use `0.0167` for 1 FPM) |
| `FRAME_TTL_SECONDS` | `2× frame interval` | Redis key TTL |
| `FFMPEG_PATH` | `ffmpeg` | Path to ffmpeg binary |

## Redis Data

- Key: `stream:{id}:frame`
- Value: JPEG bytes
- TTL: Configured via `FRAME_TTL_SECONDS`

## Frame Consumer

To consume frames and save them to disk, you must run the worker in Queue Mode and start the consumer with the **same STREAM_ID**.

1. **Start Worker (Queue Mode)**:
   ```bash
   export REDIS_QUEUE_MODE=true
   export STREAM_ID=my-stream
   export VIDEO_PATH=video.mp4
   go run ./cmd/worker
   ```

2. **Start Consumer**:
   ```bash
   export STREAM_ID=my-stream
   go run ./cmd/consumer
   ```

Frames will be saved to `./captured_frames/` (configurable via `OUTPUT_DIR`).

## Verify Frames

```bash
# Check if frame exists
redis-cli EXISTS stream:my-stream:frame

# View JPEG header (should show: ff d8 = JPEG SOI marker)
redis-cli GET stream:my-stream:frame | xxd | head -1

# Check TTL
redis-cli TTL stream:my-stream:frame
```

## Process Model

- 1 video file = 1 worker process
- Worker auto-restarts ffmpeg on failure
- Graceful shutdown on SIGINT/SIGTERM
