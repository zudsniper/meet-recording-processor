package diarize

import (
    "context"
    "github.com/zudsniper/meet-recording-processor/internal/transcribe"
)

// Silence is a minimal heuristic diarizer: alternate speakers when a gap exceeds a threshold.
// This is a placeholder and should be replaced with a proper diarization pipeline (e.g., pyannote, NeMo, or WhisperX + pyannote).
type Silence struct{}

func (Silence) AssignSpeakers(ctx context.Context, tr *transcribe.Transcript) error {
    if len(tr.Segments) == 0 {
        return nil
    }
    // If any speakers already set, do not overwrite.
    hasSpeaker := false
    for _, s := range tr.Segments { if s.Speaker != "" { hasSpeaker = true; break } }
    if hasSpeaker { return nil }

    speaker := 1
    const gapThresh = 1.5 // seconds
    for i := range tr.Segments {
        if i == 0 {
            tr.Segments[i].Speaker = speakerName(speaker)
            continue
        }
        gap := tr.Segments[i].StartSec - tr.Segments[i-1].EndSec
        if gap > gapThresh {
            // switch speaker on larger gaps
            if speaker == 1 { speaker = 2 } else { speaker = 1 }
        }
        tr.Segments[i].Speaker = speakerName(speaker)
    }
    return nil
}

func speakerName(i int) string {
    return map[int]string{1:"Speaker 1",2:"Speaker 2"}[i]
}
