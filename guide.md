# Frame Extraction Pipeline - Walkthrough

## Summary

Built a Go-based video frame extraction pipeline with the architecture:

```
Video File → ffmpeg (-re) → Go Worker → Redis
```

**Build verified**: ✅ `go build ./cmd/worker` succeeded

---

## Project Structure

```
frameextracter/
├── cmd/worker/main.go       # Entry point, orchestration
├── internal/
│   ├── config/config.go     # Environment config loading
│   ├── ffmpeg/process.go    # ffmpeg child process manager
│   ├── parser/jpeg.go       # JPEG SOI/EOI frame parser
│   └── redis/publisher.go   # Redis frame publisher
├── go.mod / go.sum          # Dependencies
├── .env.example             # Sample config
└── README.md                # Quick start guide
```

---

## Key Implementation Details

### JPEG Parser ([jpeg.go](file:///Users/swayampatil/safetyroom/frameextracter/internal/parser/jpeg.go))

- Detects SOI marker (`0xFF 0xD8`) for frame start
- Detects EOI marker (`0xFF 0xD9`) for frame end
- Streams frames via channel for non-blocking consumption

### ffmpeg Manager ([process.go](file:///Users/swayampatil/safetyroom/frameextracter/internal/ffmpeg/process.go))

- Builds command: `ffmpeg -re -i <video> -vf fps=<rate> -f image2pipe -vcodec mjpeg -`
- `-re` flag enables real-time playback simulation
- High quality output (`-q:v 2`)

### Worker ([main.go](file:///Users/swayampatil/safetyroom/frameextracter/cmd/worker/main.go))

- Auto-restarts ffmpeg on failure with configurable delay
- Graceful shutdown on SIGINT/SIGTERM
- Publishes **latest frame only** to Redis

---

## Dependencies

| Package                          | Purpose         |
| -------------------------------- | --------------- |
| `github.com/redis/go-redis/v9` | Redis client    |
| Go stdlib only                   | Everything else |

---

## Next Steps

1. **Provide sample video** to test end-to-end flow
2. **Start Redis**: `docker run -d -p 6379:6379 redis:7`
3. **Run worker**:
   ```bash
   VIDEO_PATH=/path/to/video.mp4 STREAM_ID=test go run ./cmd/worker
   ```
4. **Verify in Redis**:
   ```bash
   redis-cli GET stream:test:frame | xxd | head -1
   # Should show: ff d8 ... (JPEG SOI)
   ```