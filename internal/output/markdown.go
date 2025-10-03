package output

import (
    "fmt"
    "strings"
    "time"

    "github.com/zudsniper/meet-recording-processor/internal/transcribe"
)

type Metadata struct {
    Title     string
    Desc      string
    Attendees []string
    Source    string
    Backend   string
    Model     string
    Generated string
}

func RenderMarkdown(meta Metadata, tr transcribe.Transcript) string {
    var b strings.Builder
    // Header
    if meta.Title != "" {
        fmt.Fprintf(&b, "# %s\n\n", meta.Title)
    } else {
        b.WriteString("# Meeting Transcript\n\n")
    }
    if meta.Desc != "" {
        fmt.Fprintf(&b, "> %s\n\n", meta.Desc)
    }
    if len(meta.Attendees) > 0 {
        fmt.Fprintf(&b, "- Attendees: %s\n", strings.Join(meta.Attendees, ", "))
    }
    if meta.Source != "" {
        fmt.Fprintf(&b, "- Source: `%s`\n", meta.Source)
    }
    if meta.Backend != "" {
        fmt.Fprintf(&b, "- Backend: `%s`\n", meta.Backend)
    }
    if meta.Model != "" {
        fmt.Fprintf(&b, "- Model: `%s`\n", meta.Model)
    }
    if meta.Generated != "" {
        fmt.Fprintf(&b, "- Generated: %s\n", meta.Generated)
    }
    if tr.Duration > 0 {
        fmt.Fprintf(&b, "- Duration: %s\n", tr.Duration.Truncate(time.Second))
    }
    b.WriteString("\n---\n\n")

    // Body
    for _, s := range tr.Segments {
        ts := ""
        if s.EndSec > 0 {
            ts = fmt.Sprintf("[%s-%s] ", secToTS(s.StartSec), secToTS(s.EndSec))
        }
        spk := ""
        if s.Speaker != "" {
            spk = s.Speaker + ": "
        }
        fmt.Fprintf(&b, "%s%s%s\n\n", ts, spk, strings.TrimSpace(s.Text))
    }
    return b.String()
}

func secToTS(sec float64) string {
    d := time.Duration(sec*1000) * time.Millisecond
    h := int(d.Hours())
    m := int(d.Minutes()) % 60
    s := int(d.Seconds()) % 60
    if h > 0 { return fmt.Sprintf("%02d:%02d:%02d", h, m, s) }
    return fmt.Sprintf("%02d:%02d", m, s)
}
