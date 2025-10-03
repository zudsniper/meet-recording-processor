package transcribe

import (
    "context"
    "embed"
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"
)

//go:embed assets/faster_whisper.py
var fwScript []byte

type fasterWhisperBackend struct {
    model  string
    device string // auto|cpu|cuda
}

func NewFasterWhisperBackend(model, device string) Backend {
    return &fasterWhisperBackend{model: model, device: device}
}

type fwOut struct {
    Language string `json:"language"`
    Duration float64 `json:"duration"`
    Segments []struct{
        Start float64 `json:"start"`
        End   float64 `json:"end"`
        Text  string  `json:"text"`
    } `json:"segments"`
}

func (f *fasterWhisperBackend) Transcribe(ctx context.Context, audioPath string) (Transcript, error) {
    // Write embedded script to temp
    dir := os.TempDir()
    scriptPath := filepath.Join(dir, "mrp_faster_whisper.py")
    if err := os.WriteFile(scriptPath, fwScript, 0o755); err != nil {
        return Transcript{}, fmt.Errorf("write helper script: %w", err)
    }
    defer os.Remove(scriptPath)

    device := f.device
    if device == "" { device = "auto" }
    // Call python3 helper
    py := os.Getenv("MRP_PY")
    if py == "" {
        py = "python3"
    }
    cmd := exec.CommandContext(ctx, py, scriptPath, "--audio", audioPath, "--model", f.model, "--device", device)
    cmd.Env = os.Environ()
    out, err := cmd.Output()
    if err != nil {
        // try to capture stderr
        if ee, ok := err.(*exec.ExitError); ok {
            return Transcript{}, fmt.Errorf("faster-whisper failed: %s", strings.TrimSpace(string(ee.Stderr)))
        }
        return Transcript{}, fmt.Errorf("run helper: %w", err)
    }
    var parsed fwOut
    if err := json.Unmarshal(out, &parsed); err != nil {
        return Transcript{}, fmt.Errorf("parse helper output: %w\n%s", err, string(out))
    }
    tr := Transcript{Language: parsed.Language, Duration: time.Duration(parsed.Duration*float64(time.Second))}
    for _, s := range parsed.Segments {
        tr.Segments = append(tr.Segments, Segment{StartSec: s.Start, EndSec: s.End, Text: strings.TrimSpace(s.Text)})
    }
    return tr, nil
}
