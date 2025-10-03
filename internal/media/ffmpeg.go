package media

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

// ExtractAudio uses ffmpeg to extract mono 16kHz WAV from a video.
// Returns the path to the extracted audio file.
func ExtractAudio(ctx context.Context, videoPath string, tmpDir string) (string, error) {
    if tmpDir == "" {
        tmpDir = os.TempDir()
    }
    base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
    out := filepath.Join(tmpDir, base+"_audio_16k.wav")

    // ffmpeg -y -i input -ac 1 -ar 16000 -f wav output
    cmd := exec.CommandContext(ctx, "ffmpeg",
        "-y", "-i", videoPath,
        "-ac", "1", "-ar", "16000",
        "-f", "wav",
        out,
    )
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("ffmpeg: %w", err)
    }
    return out, nil
}
