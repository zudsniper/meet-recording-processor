package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/zudsniper/meet-recording-processor/internal/diarize"
    "github.com/zudsniper/meet-recording-processor/internal/media"
    "github.com/zudsniper/meet-recording-processor/internal/output"
    "github.com/zudsniper/meet-recording-processor/internal/transcribe"
)

const (
    colorReset  = "\033[0m"
    colorGreen  = "\033[32m"
    colorYellow = "\033[33m"
    colorBlue   = "\033[34m"
    colorRed    = "\033[31m"
)

func info(msg string, a ...any) {
    fmt.Fprintf(os.Stderr, colorBlue+"[info] "+colorReset+msg+"\n", a...)
}

func warn(msg string, a ...any) {
    fmt.Fprintf(os.Stderr, colorYellow+"[warn] "+colorReset+msg+"\n", a...)
}

func ok(msg string, a ...any) {
    fmt.Fprintf(os.Stderr, colorGreen+"[ok] "+colorReset+msg+"\n", a...)
}

func fail(msg string, a ...any) {
    fmt.Fprintf(os.Stderr, colorRed+"[error] "+colorReset+msg+"\n", a...)
}

type stringSlice []string

func (s *stringSlice) String() string { return strings.Join(*s, ",") }
func (s *stringSlice) Set(v string) error {
    if v == "" {
        return nil
    }
    parts := strings.Split(v, ",")
    for _, p := range parts {
        p = strings.TrimSpace(p)
        if p != "" {
            *s = append(*s, p)
        }
    }
    return nil
}

func main() {
    var (
        inPath    string
        outPath   string
        backend   string
        model     string
        tmpDir    string
        diarizer  string
        eventTitle string
        eventDesc  string
        attendees stringSlice

        openaiAPIKey string
        openaiModel  string

        cfAccountID string
        cfAPIToken  string
        cfModel     string

        localModel   string
        localDevice  string
    )

    flag.StringVar(&inPath, "input", "", "Input video file path (-i)")
    flag.StringVar(&inPath, "i", "", "Input video file path")
    flag.StringVar(&outPath, "output", "", "Output transcript markdown file (-o)")
    flag.StringVar(&outPath, "o", "", "Output transcript markdown file")
    flag.StringVar(&backend, "backend", "openai", "Transcription backend: openai|cloudflare|local")
    flag.StringVar(&model, "model", "", "Generic model name override (backend-specific)")
    flag.StringVar(&tmpDir, "tmpdir", "", "Temporary working directory (default system temp)")
    flag.StringVar(&diarizer, "diarization", "none", "Diarization: none|silence")

    flag.StringVar(&eventTitle, "title", "", "Event title metadata")
    flag.StringVar(&eventDesc, "description", "", "Event description metadata")
    flag.Var(&attendees, "attendee", "Attendee name (repeatable or comma-separated)")

    flag.StringVar(&openaiAPIKey, "openai-api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key (or set OPENAI_API_KEY)")
    flag.StringVar(&openaiModel, "openai-model", "gpt-4o-mini-transcribe", "OpenAI transcription model")

    flag.StringVar(&cfAccountID, "cf-account-id", os.Getenv("CF_ACCOUNT_ID"), "Cloudflare Account ID (or CF_ACCOUNT_ID)")
    flag.StringVar(&cfAPIToken, "cf-api-token", os.Getenv("CF_API_TOKEN"), "Cloudflare API Token (or CF_API_TOKEN)")
    flag.StringVar(&cfModel, "cf-model", "@cf/openai/whisper", "Cloudflare AI model identifier")

    flag.StringVar(&localModel, "local-model", "base.en", "faster-whisper model name or path (e.g., base.en, medium, or local path)")
    flag.StringVar(&localDevice, "local-device", "auto", "Device for local model: auto|cpu|cuda (default respects MRP_DEFAULT_LOCAL_DEVICE)")

    flag.Parse()

    if inPath == "" {
        fail("missing --input/-i video path")
        os.Exit(2)
    }
    if outPath == "" {
        base := strings.TrimSuffix(filepath.Base(inPath), filepath.Ext(inPath))
        outPath = base + ".md"
    }

    // Prepare context
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
    defer cancel()

    // Step 1: extract audio
    info("Extracting audio via ffmpeg...")
    audioPath, err := media.ExtractAudio(ctx, inPath, tmpDir)
    if err != nil {
        fail("audio extraction failed: %v", err)
        os.Exit(1)
    }
    ok("Audio ready: %s", audioPath)

    // Step 2: pick backend
    var be transcribe.Backend
    switch strings.ToLower(backend) {
    case "openai":
        if openaiAPIKey == "" {
            fail("OpenAI backend selected but API key is missing")
            os.Exit(1)
        }
        if model != "" {
            openaiModel = model
        }
        be = transcribe.NewOpenAIBackend(openaiAPIKey, openaiModel)
    case "cloudflare":
        if cfAccountID == "" || cfAPIToken == "" {
            fail("Cloudflare backend requires cf-account-id and cf-api-token")
            os.Exit(1)
        }
        if model != "" {
            cfModel = model
        }
        be = transcribe.NewCloudflareBackend(cfAccountID, cfAPIToken, cfModel)
    case "local":
        if model != "" {
            localModel = model
        }
        // Respect env default for device if user did not choose
        if strings.ToLower(localDevice) == "auto" {
            if envDev := strings.ToLower(strings.TrimSpace(os.Getenv("MRP_DEFAULT_LOCAL_DEVICE"))); envDev == "cpu" || envDev == "cuda" {
                localDevice = envDev
            } else if envDev2 := strings.ToLower(strings.TrimSpace(os.Getenv("MRP_LOCAL_DEVICE"))); envDev2 == "cpu" || envDev2 == "cuda" {
                localDevice = envDev2
            }
        }
        if py, err := ensureLocalFasterWhisper(ctx); err != nil {
            fail("local backend setup failed: %v", err)
            os.Exit(1)
        } else if py != "" {
            os.Setenv("MRP_PY", py)
        }
        be = transcribe.NewFasterWhisperBackend(localModel, localDevice)
    default:
        fail("unknown backend: %s", backend)
        os.Exit(2)
    }

    // Step 3: transcribe
    info("Transcribing using %s backend...", backend)
    tr, err := be.Transcribe(ctx, audioPath)
    if err != nil {
        fail("transcription failed: %v", err)
        os.Exit(1)
    }
    ok("Transcription done: %d segments", len(tr.Segments))

    // Step 4: diarization (minimal option)
    var diarizerImpl diarize.Diarizer
    switch strings.ToLower(diarizer) {
    case "none":
        diarizerImpl = diarize.Noop{}
    case "silence":
        diarizerImpl = diarize.Silence{}
    default:
        fail("unknown diarization mode: %s", diarizer)
        os.Exit(2)
    }
    info("Applying diarization: %s...", diarizer)
    if err := diarizerImpl.AssignSpeakers(ctx, &tr); err != nil {
        warn("diarization skipped/failed: %v", err)
    } else {
        ok("Diarization applied")
    }

    // Step 5: render markdown
    meta := output.Metadata{
        Title:     eventTitle,
        Desc:      eventDesc,
        Attendees: []string(attendees),
        Source:    inPath,
        Backend:   backend,
        Model:     func() string { if model != "" { return model }; return modelFromBackend(backend, openaiModel, cfModel, localModel) }(),
        Generated: time.Now().Format(time.RFC3339),
    }

    md := output.RenderMarkdown(meta, tr)
    if err := os.WriteFile(outPath, []byte(md), 0o644); err != nil {
        fail("writing output: %v", err)
        os.Exit(1)
    }
    ok("Wrote %s", outPath)
}

func modelFromBackend(backend, openaiModel, cfModel, localModel string) string {
    switch backend {
    case "openai":
        return openaiModel
    case "cloudflare":
        return cfModel
    case "local":
        return localModel
    default:
        return ""
    }
}

// ensureLocalFasterWhisper ensures a functional Python environment with faster-whisper installed.
// It attempts to use $MRP_PY, then ~/.mrp/venv, or creates a new venv and installs packages.
// Returns the python interpreter path to use (may be empty if unchanged).
func ensureLocalFasterWhisper(ctx context.Context) (string, error) {
    // If MRP_PY provided and usable, keep it
    if py := strings.TrimSpace(os.Getenv("MRP_PY")); py != "" {
        if err := fwImportCheck(ctx, py); err == nil {
            return py, nil
        }
    }

    home, err := os.UserHomeDir()
    if err != nil { return "", fmt.Errorf("home dir: %w", err) }
    venvDir := filepath.Join(home, ".mrp", "venv")
    pyPath := filepath.Join(venvDir, "bin", "python")

    // If venv exists and faster-whisper is installed, use it
    if _, err := os.Stat(pyPath); err == nil {
        if err := fwImportCheck(ctx, pyPath); err == nil {
            return pyPath, nil
        }
    }

    // Otherwise, bootstrap venv and install faster-whisper
    // Require python3 to be available
    if _, err := exec.LookPath("python3"); err != nil {
        return "", fmt.Errorf("python3 not found; please run scripts/install.sh or install Python 3")
    }

    info("Setting up local faster-whisper environment (venv + pip install)...")
    if err := os.MkdirAll(filepath.Dir(venvDir), 0o755); err != nil {
        return "", fmt.Errorf("mkdir: %w", err)
    }
    if _, err := os.Stat(venvDir); os.IsNotExist(err) {
        if err := execCmd(ctx, "python3", "-m", "venv", venvDir); err != nil {
            return "", fmt.Errorf("create venv: %w", err)
        }
    }
    if err := execCmd(ctx, pyPath, "-m", "pip", "install", "--upgrade", "pip"); err != nil {
        return "", fmt.Errorf("upgrade pip: %w", err)
    }
    if err := execCmd(ctx, pyPath, "-m", "pip", "install", "faster-whisper"); err != nil {
        return "", fmt.Errorf("install faster-whisper: %w", err)
    }
    if err := fwImportCheck(ctx, pyPath); err != nil {
        return "", fmt.Errorf("verify faster-whisper: %w", err)
    }
    ok("Local faster-whisper ready")
    return pyPath, nil
}

func fwImportCheck(ctx context.Context, py string) error {
    cmd := exec.CommandContext(ctx, py, "-c", "import faster_whisper; print('ok')")
    cmd.Env = os.Environ()
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func execCmd(ctx context.Context, name string, args ...string) error {
    cmd := exec.CommandContext(ctx, name, args...)
    cmd.Env = os.Environ()
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
