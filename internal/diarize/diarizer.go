package diarize

import (
    "context"
    "github.com/zudsniper/meet-recording-processor/internal/transcribe"
)

// Diarizer assigns speaker labels to transcript segments.
type Diarizer interface {
    AssignSpeakers(ctx context.Context, tr *transcribe.Transcript) error
}

// Noop leaves speakers empty.
type Noop struct{}

func (Noop) AssignSpeakers(ctx context.Context, tr *transcribe.Transcript) error { return nil }
