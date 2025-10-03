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

// Cloudflare Workers AI backend.
// POST https://api.cloudflare.com/client/v4/accounts/{account_id}/ai/run/{model}
// With bearer API token.
type cloudflareBackend struct {
    accountID string
    apiToken  string
    model     string
}

func NewCloudflareBackend(accountID, apiToken, model string) Backend {
    return &cloudflareBackend{accountID: accountID, apiToken: apiToken, model: model}
}

type cfResp struct {
    Success bool            `json:"success"`
    Errors  []any           `json:"errors"`
    Result  json.RawMessage `json:"result"`
}

type cfWhisperResult struct {
    Text string `json:"text"`
}

func (c *cloudflareBackend) Transcribe(ctx context.Context, audioPath string) (Transcript, error) {
    f, err := os.Open(audioPath)
    if err != nil {
        return Transcript{}, err
    }
    defer f.Close()

    var body bytes.Buffer
    mw := multipart.NewWriter(&body)
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

    url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/run/%s", c.accountID, c.model)
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
    if err != nil {
        return Transcript{}, err
    }
    req.Header.Set("Authorization", "Bearer "+c.apiToken)
    req.Header.Set("Content-Type", mw.FormDataContentType())

    hc := &http.Client{Timeout: 60 * time.Minute}
    resp, err := hc.Do(req)
    if err != nil {
        return Transcript{}, err
    }
    defer resp.Body.Close()
    if resp.StatusCode >= 300 {
        b, _ := io.ReadAll(resp.Body)
        return Transcript{}, fmt.Errorf("cloudflare http %d: %s", resp.StatusCode, string(b))
    }
    var cr cfResp
    if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
        return Transcript{}, err
    }
    if !cr.Success {
        return Transcript{}, fmt.Errorf("cloudflare response not successful: %s", string(cr.Result))
    }
    var wr cfWhisperResult
    if err := json.Unmarshal(cr.Result, &wr); err != nil {
        // Some models might return different formats; fallback to raw string
        // Attempt to decode as { "text": "..." } first; otherwise make a best effort
        return Transcript{}, fmt.Errorf("cloudflare unexpected result: %w", err)
    }
    t := Transcript{Language: "", Segments: []Segment{{StartSec: 0, EndSec: 0, Text: wr.Text}}, Duration: 0}
    return t, nil
}

