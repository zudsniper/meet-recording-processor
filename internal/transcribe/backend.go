package transcribe

import (
    "context"
    "time"
)

// Segment represents a portion of transcribed audio.
type Segment struct {
    StartSec float64
    EndSec   float64
    Text     string
    Speaker  string // optional; to be filled by diarization
}

// Transcript bundles the segments.
type Transcript struct {
    Language string
    Segments []Segment
    Duration time.Duration
}

// Backend is a pluggable transcription backend.
type Backend interface {
    Transcribe(ctx context.Context, audioPath string) (Transcript, error)
}

