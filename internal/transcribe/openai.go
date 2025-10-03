package transcribe

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
    "path/filepath"
    "time"
)

// OpenAI speech-to-text via audio.transcriptions
type openAIBackend struct {
    apiKey string
    model  string
}

func NewOpenAIBackend(apiKey, model string) Backend {
    return &openAIBackend{apiKey: apiKey, model: model}
}

type openAIResp struct {
    Text string `json:"text"`
}

func (o *openAIBackend) Transcribe(ctx context.Context, audioPath string) (Transcript, error) {
    // Build multipart payload
    f, err := os.Open(audioPath)
    if err != nil {
        return Transcript{}, err
    }
    defer f.Close()

    var body bytes.Buffer
    mw := multipart.NewWriter(&body)

    if err := mw.WriteField("model", o.model); err != nil {
        return Transcript{}, err
    }
    // response_format: json? But default returns text. Use verbose_json not supported here.
    // We'll request plain text and post-process as a single segment.

    fw, err := mw.CreateFormFile("file", filepath.Base(audioPath))
    if err != nil {
        return Transcript{}, err
    }
    if _, err := io.Copy(fw, f); err != nil {
        return Transcript{}, err
    }
    if err := mw.Close(); err != nil {
        return Transcript{}, err
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/audio/transcriptions", &body)
    if err != nil {
        return Transcript{}, err
    }
    req.Header.Set("Authorization", "Bearer "+o.apiKey)
    req.Header.Set("Content-Type", mw.FormDataContentType())

    hc := &http.Client{Timeout: 60 * time.Minute}
    resp, err := hc.Do(req)
    if err != nil {
        return Transcript{}, err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return Transcript{}, fmt.Errorf("openai http %d: %s", resp.StatusCode, string(b))
    }
    var or openAIResp
    if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
        return Transcript{}, err
    }
    // We didn't get segment timings. For now, return as a single segment.
    t := Transcript{Language: "", Segments: []Segment{{StartSec: 0, EndSec: 0, Text: or.Text}}, Duration: 0}
    return t, nil
}

