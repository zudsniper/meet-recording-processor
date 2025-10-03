# Meet Recording Processor (CLI)

A pluggable Go CLI that extracts audio from meeting recordings (via `ffmpeg`) and transcribes them with one of multiple backends:

- OpenAI Audio Transcriptions API
- Cloudflare Workers AI (`@cf/openai/whisper`)
- Local faster-whisper (GPU-friendly; via a small embedded Python helper)

Designed to evolve into an automated service later (e.g., trigger on Google Drive upload), while remaining simple and fast locally today.

## Requirements

- `ffmpeg` in PATH
- OpenAI backend: `OPENAI_API_KEY` env var (or `--openai-api-key`)
- Cloudflare backend: `CF_ACCOUNT_ID` and `CF_API_TOKEN` env vars (or flags)
- Local backend: Python 3; the installer sets up a venv at `~/.mrp/venv` and installs `faster-whisper`, exporting `MRP_PY` to that interpreter.

## Build / Install

Build from source:

```
go build -o mrp ./cmd/mrp
```

Installer (interactive):

```
curl -fsSL https://raw.githubusercontent.com/zudsniper/meet-recording-processor/main/scripts/install.sh | bash
```

Non-interactive (accept defaults):

```
curl -fsSL https://raw.githubusercontent.com/zudsniper/meet-recording-processor/main/scripts/install.sh | bash -s -- -y
```

## Usage

Common flags:

- `--input, -i`: path to video file
- `--output, -o`: output markdown file (default: `<video-name>.md`)
- `--backend`: `openai` (default) | `cloudflare` | `local`
- `--model`: model override (backend-specific); for local, prefer `--local-model`
- `--tmpdir`: temp directory for intermediate audio
- `--diarization`: `none` (default) | `silence` (heuristic alternating speakers on gaps)
- Metadata: `--title`, `--description`, `--attendee` (repeatable)

Local faster-whisper specific:

- `--local-model` (e.g., `base.en`, `small`, or local path)
- `--local-device` `auto|cpu|cuda` (default `auto`)
- `MRP_PY` env var controls which Python interpreter is used (installer sets it to the venv python).

OpenAI-specific:

- `--openai-api-key` (or env `OPENAI_API_KEY`)
- `--openai-model` (default `gpt-4o-mini-transcribe`)

Cloudflare-specific:

- `--cf-account-id` (or env `CF_ACCOUNT_ID`)
- `--cf-api-token` (or env `CF_API_TOKEN`)
- `--cf-model` (default `@cf/openai/whisper`)

### Quick Examples

Local (CPU/GPU auto):

```
mrp -i ~/Downloads/Top8Meeting.mp4 --backend local -o Top8Meeting.md \
    --title "Top 8 Meeting" --description "Weekly status"
```

Local with CUDA GPU:

```
mrp -i ~/Downloads/Top8Meeting.mp4 --backend local --local-device cuda \
    --local-model base.en -o Top8Meeting.md
```

OpenAI (needs `OPENAI_API_KEY`):

```
mrp -i meeting.mp4 --backend openai --model gpt-4o-mini-transcribe -o transcript.md
```

Cloudflare (needs `CF_ACCOUNT_ID` and `CF_API_TOKEN`):

```
mrp -i meeting.mp4 --backend cloudflare --cf-model @cf/openai/whisper -o transcript.md
```

Diarization (simple heuristic):

```
mrp -i meeting.mp4 --backend local --diarization silence -o transcript.md
```

## Notes on Diarization

This initial version includes a minimal `--diarization silence` mode that alternates speakers when a gap between segments exceeds ~1.5s. It is only a placeholder. For high-quality diarization, consider:

- WhisperX + pyannote.audio for alignment + diarization
- NVIDIA NeMo diarization pipeline (speaker embeddings + clustering)
- Modern E2E diarization approaches (e.g., EEND variants)

These can be integrated later as a separate backend/module without changing the CLI surface.

## Troubleshooting

- `mrp: command not found`:
  - Ensure `/usr/local/bin` or `$HOME/.local/bin` is in your `PATH`.
- `ffmpeg not found`:
  - Re-run the installer; it will install ffmpeg via your package manager.
- Local backend errors about Python or faster-whisper:
  - The installer creates a venv at `~/.mrp/venv` and sets `MRP_PY` to that interpreter. Ensure your shell loaded `~/.mrp.env` or run `export MRP_PY=$HOME/.mrp/venv/bin/python`.
- CUDA not used when expected:
  - Add `--local-device cuda`. Ensure CUDA toolkit and compatible drivers are installed.

## Future Direction

- Expose a long-running service and trigger from Google Drive/Meet callbacks
- Post-process transcript with prompts to file issues or summaries
- Persist results and metadata, add speaker mapping using attendee names
