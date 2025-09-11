package local

import (
    "context"
    "os/exec"
    "strconv"
    "strings"

    "github.com/faraz525/home-music-server/backend/internal/media/metadata"
)

type FFProbeExtractor struct{}

func New() *FFProbeExtractor { return &FFProbeExtractor{} }

func (e *FFProbeExtractor) Extract(ctx context.Context, fullPath string) (*metadata.AudioMetadata, error) {
    // Minimal duration extraction via ffprobe; extend as needed
    out, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=nk=1:nw=1", fullPath).Output()
    if err != nil {
        return &metadata.AudioMetadata{}, nil
    }
    s := strings.TrimSpace(string(out))
    dur, _ := strconv.ParseFloat(s, 64)
    return &metadata.AudioMetadata{DurationSeconds: dur}, nil
}

