# Meet Recording Processor (CLI)

A pluggable Go CLI to extract audio from a meeting recording and transcribe it via one of several backends (OpenAI, Cloudflare Workers AI, local faster-whisper). Minimal, expandable design for future automation.

## Requirements

- `ffmpeg` in PATH
- For OpenAI backend: `OPENAI_API_KEY` env var (or `--openai-api-key`)
- For Cloudflare backend: `CF_ACCOUNT_ID` and `CF_API_TOKEN` env vars (or flags)
- For local backend: Python 3; installer sets up a venv at `~/.mrp/venv` and installs `faster-whisper` there, exporting `MRP_PY`.

## Build

```
go build -o mrp ./cmd/mrp
```

## Usage

```
./mrp --input meeting.mp4 --backend openai -o transcript.md \
  --title "Weekly Sync" --description "Sprint planning" --attendee "Alice" --attendee "Bob"

./mrp -i meeting.mp4 --backend cloudflare --cf-model @cf/openai/whisper -o transcript.md

./mrp -i meeting.mp4 --backend local --local-model base.en -o transcript.md
```

Common flags:

- `--input, -i`: path to video file
- `--output, -o`: output markdown file (default: `<video-name>.md`)
- `--backend`: `openai` (default) | `cloudflare` | `local`
- `--model`: model override (backend-specific); for local, prefer `--local-model`
- `--tmpdir`: temp directory for intermediate audio
- `--diarization`: `none` (default) | `silence` (heuristic alternating speakers on gaps)
- Metadata: `--title`, `--description`, `--attendee` (repeatable)

OpenAI-specific:

- `--openai-api-key` (or env `OPENAI_API_KEY`)
- `--openai-model` (default `gpt-4o-mini-transcribe`)

Cloudflare-specific:

- `--cf-account-id` (or env `CF_ACCOUNT_ID`)
- `--cf-api-token` (or env `CF_API_TOKEN`)
- `--cf-model` (default `@cf/openai/whisper`)

Local faster-whisper specific:

- `--local-model` (e.g., `base.en`, `small`, or local path)
- `--local-device` `auto|cpu|cuda` (default `auto`)
- `MRP_PY` env var controls which Python interpreter is used for local transcription (installer sets it to the venv python).

## Notes on Diarization

This initial version includes a minimal `--diarization silence` mode that alternates speakers when a gap between segments exceeds ~1.5s. It is only a placeholder. For high-quality diarization, consider:

- WhisperX + pyannote.audio for alignment + diarization
- NVIDIA NeMo diarization pipeline (speaker embeddings + clustering)
- Modern E2E diarization approaches (e.g., EEND variants)

These can be integrated later as a separate backend/module without changing the CLI surface.

## Future Direction

- Expose a long-running service and trigger from Google Drive/Meet callbacks
- Post-process transcript with prompts to file issues or summaries
- Persist results and metadata, add speaker mapping using attendee names
Or run the installer (interactive):

```
curl -fsSL https://raw.githubusercontent.com/zudsniper/meet-recording-processor/main/scripts/install.sh | bash
```

Non-interactive (accept defaults):

```
curl -fsSL https://raw.githubusercontent.com/zudsniper/meet-recording-processor/main/scripts/install.sh | bash -s -- -y
```
