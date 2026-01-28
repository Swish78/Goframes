package ffmpeg

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strconv"
	"sync"
)

// Process manages an ffmpeg child process
type Process struct {
	videoPath  string
	frameRate  float64
	ffmpegPath string

	cmd    *exec.Cmd
	stdout io.ReadCloser
	mu     sync.Mutex
}

// NewProcess creates a new ffmpeg process manager
func NewProcess(videoPath string, frameRate float64, ffmpegPath string) *Process {
	if ffmpegPath == "" {
		ffmpegPath = "ffmpeg"
	}
	return &Process{
		videoPath:  videoPath,
		frameRate:  frameRate,
		ffmpegPath: ffmpegPath,
	}
}

// Start launches the ffmpeg process and returns its stdout
func (p *Process) Start(ctx context.Context) (io.Reader, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Build ffmpeg command:
	// ffmpeg -re -i <video> -vf fps=<rate> -f image2pipe -vcodec mjpeg -
	args := []string{
		"-re",                                      // Real-time playback
		"-i", p.videoPath,                          // Input file
		"-vf", fmt.Sprintf("fps=%s", strconv.FormatFloat(p.frameRate, 'f', -1, 64)), // Frame rate filter
		"-f", "image2pipe",                         // Output format: pipe
		"-vcodec", "mjpeg",                         // JPEG codec
		"-q:v", "2",                                // Quality (2 = high quality)
		"-",                                        // Output to stdout
	}

	p.cmd = exec.CommandContext(ctx, p.ffmpegPath, args...)

	// Get stdout pipe
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	p.stdout = stdout

	// Start the process
	if err := p.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	log.Printf("ffmpeg started: pid=%d, video=%s, fps=%.2f",
		p.cmd.Process.Pid, p.videoPath, p.frameRate)

	return stdout, nil
}

// Wait waits for the ffmpeg process to exit
func (p *Process) Wait() error {
	p.mu.Lock()
	cmd := p.cmd
	p.mu.Unlock()

	if cmd == nil {
		return nil
	}
	return cmd.Wait()
}

// Stop terminates the ffmpeg process
func (p *Process) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	log.Printf("stopping ffmpeg: pid=%d", p.cmd.Process.Pid)

	if err := p.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill ffmpeg: %w", err)
	}

	return nil
}

// Running returns true if the ffmpeg process is currently running
func (p *Process) Running() bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return false
	}

	// ProcessState is set after Wait() completes
	return p.cmd.ProcessState == nil
}
