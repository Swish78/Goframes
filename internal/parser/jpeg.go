package parser

import (
	"bufio"
	"context"
	"io"
)

// JPEG markers
const (
	markerPrefix = 0xFF
	markerSOI    = 0xD8 // Start Of Image
	markerEOI    = 0xD9 // End Of Image
)

// Frame represents a complete JPEG frame
type Frame struct {
	Data []byte
}

// Parser reads JPEG frames from an io.Reader by detecting SOI/EOI markers
type Parser struct {
	reader *bufio.Reader
}

// NewParser creates a new JPEG parser
func NewParser(r io.Reader) *Parser {
	return &Parser{
		reader: bufio.NewReaderSize(r, 64*1024), // 64KB buffer
	}
}

// ParseFrames reads frames from the input and sends them to the output channel.
// It closes the channel when the context is cancelled or the input is exhausted.
func (p *Parser) ParseFrames(ctx context.Context, frames chan<- Frame) error {
	defer close(frames)

	var frameBuffer []byte
	inFrame := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		b, err := p.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if inFrame {
			frameBuffer = append(frameBuffer, b)

			// Check for EOI marker (end of frame)
			if len(frameBuffer) >= 2 {
				prev := frameBuffer[len(frameBuffer)-2]
				if prev == markerPrefix && b == markerEOI {
					// Complete frame found
					frameCopy := make([]byte, len(frameBuffer))
					copy(frameCopy, frameBuffer)

					select {
					case frames <- Frame{Data: frameCopy}:
					case <-ctx.Done():
						return ctx.Err()
					}

					// Reset for next frame
					frameBuffer = frameBuffer[:0]
					inFrame = false
				}
			}
		} else {
			// Looking for SOI marker
			if len(frameBuffer) == 1 && frameBuffer[0] == markerPrefix && b == markerSOI {
				// Found SOI, start collecting frame
				frameBuffer = append(frameBuffer, b)
				inFrame = true
			} else if b == markerPrefix {
				// Potential start of marker
				frameBuffer = []byte{b}
			} else {
				// Not a marker, discard
				frameBuffer = frameBuffer[:0]
			}
		}
	}
}
