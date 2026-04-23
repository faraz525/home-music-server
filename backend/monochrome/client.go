package monochrome

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Quality levels accepted by the upstream /track/ endpoint.
const (
	QualityHiRes    = "HI_RES_LOSSLESS"
	QualityLossless = "LOSSLESS"
	QualityHigh     = "HIGH"
	QualityLow      = "LOW"
)

// manifestTypeBTS is the mime type for plain lossless FLAC (TIDAL BTS container).
const manifestTypeBTS = "application/vnd.tidal.bts"

// defaultUserAgent mimics a common desktop Chrome. Several community mirrors
// sit behind Cloudflare and 403 requests from Go's default `Go-http-client/1.1`
// UA — this gets us through.
const defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

// Client talks to one or more monochrome.tf-compatible hifi-api instances.
// Multiple base URLs are tried in order — backends get TIDAL-banned regularly,
// so the monochrome.tf website itself juggles ~10 mirrors. Zero value is not
// usable — use NewClient.
type Client struct {
	baseURLs  []string
	http      *http.Client
	userAgent string
}

// NewClient returns a client with failover across baseURLs. Hosts are tried in
// order per request; the first to return a valid response wins. Empty strings
// are filtered; trailing slashes trimmed. timeout bounds each individual HTTP
// request — use >= 60s to accommodate CDN FLAC downloads.
func NewClient(baseURLs []string, timeout time.Duration) *Client {
	cleaned := make([]string, 0, len(baseURLs))
	for _, u := range baseURLs {
		u = strings.TrimRight(strings.TrimSpace(u), "/")
		if u != "" {
			cleaned = append(cleaned, u)
		}
	}
	return &Client{
		baseURLs:  cleaned,
		http:      &http.Client{Timeout: timeout},
		userAgent: defaultUserAgent,
	}
}

// Hosts returns the configured base URLs (for logging).
func (c *Client) Hosts() []string {
	out := make([]string, len(c.baseURLs))
	copy(out, c.baseURLs)
	return out
}

// TrackMatch is a normalized catalog hit returned by Search.
type TrackMatch struct {
	TidalID     int
	ISRC        string
	Title       string
	Artists     []string
	Album       string
	DurationSec int
	Quality     string // upstream audioQuality: HI_RES_LOSSLESS, LOSSLESS, HIGH, LOW
}

// StreamInfo describes a resolved, signed CDN URL for the audio bytes.
type StreamInfo struct {
	URL      string
	Quality  string // quality upstream actually served — may be below what was requested
	MimeType string
	Codec    string
}

// Search runs a free-text query against the TIDAL catalog and returns up to
// `limit` matches. The upstream /search/ endpoint accepts only free-text via
// `s=` — there is no ISRC query param, so ISRC filtering must be done by the
// caller after reading TrackMatch.ISRC on each result.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]TrackMatch, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("empty query")
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	q := url.Values{
		"s":     {query},
		"limit": {fmt.Sprintf("%d", limit)},
	}
	return tryAllHosts(c.baseURLs, func(base string) ([]TrackMatch, error) {
		return c.searchOne(ctx, base, q)
	})
}

func (c *Client) searchOne(ctx context.Context, base string, q url.Values) ([]TrackMatch, error) {
	u := base + "/search/?" + q.Encode()
	body, err := c.fetchJSON(ctx, u)
	if err != nil {
		return nil, err
	}
	var parsed searchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	matches := make([]TrackMatch, 0, len(parsed.Data.Items))
	for _, it := range parsed.Data.Items {
		artists := make([]string, 0, len(it.Artists))
		for _, a := range it.Artists {
			artists = append(artists, a.Name)
		}
		matches = append(matches, TrackMatch{
			TidalID:     it.ID,
			ISRC:        it.ISRC,
			Title:       it.Title,
			Artists:     artists,
			Album:       it.Album.Title,
			DurationSec: it.Duration,
			Quality:     it.AudioQuality,
		})
	}
	return matches, nil
}

type searchResponse struct {
	Data struct {
		Items []struct {
			ID           int    `json:"id"`
			Title        string `json:"title"`
			Duration     int    `json:"duration"`
			ISRC         string `json:"isrc"`
			AudioQuality string `json:"audioQuality"`
			Artists      []struct {
				Name string `json:"name"`
			} `json:"artists"`
			Album struct {
				Title string `json:"title"`
			} `json:"album"`
		} `json:"items"`
	} `json:"data"`
}

// GetStreamInfo resolves a tidal track ID into a signed CDN URL. quality is one
// of the Quality* constants; if upstream can't deliver the requested tier it
// silently downgrades — the returned StreamInfo.Quality reflects what was
// actually served.
//
// Errors on DRM-protected / Atmos / MQA responses (DASH manifests, non-NONE
// encryption) — caller should fall back to another download source.
func (c *Client) GetStreamInfo(ctx context.Context, tidalID int, quality string) (*StreamInfo, error) {
	if tidalID <= 0 {
		return nil, fmt.Errorf("invalid tidal id: %d", tidalID)
	}
	if quality == "" {
		quality = QualityLossless
	}
	return tryAllHosts(c.baseURLs, func(base string) (*StreamInfo, error) {
		return c.streamOne(ctx, base, tidalID, quality)
	})
}

func (c *Client) streamOne(ctx context.Context, base string, tidalID int, quality string) (*StreamInfo, error) {
	u := fmt.Sprintf("%s/track/?id=%d&quality=%s", base, tidalID, url.QueryEscape(quality))
	body, err := c.fetchJSON(ctx, u)
	if err != nil {
		return nil, err
	}
	var parsed playbackResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode playback response: %w", err)
	}
	if parsed.Data.ManifestMimeType != manifestTypeBTS {
		return nil, fmt.Errorf("unsupported manifest type %q (DRM/DASH not supported)", parsed.Data.ManifestMimeType)
	}
	raw, err := base64.StdEncoding.DecodeString(parsed.Data.Manifest)
	if err != nil {
		return nil, fmt.Errorf("base64 decode manifest: %w", err)
	}
	var manifest manifestContent
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest json: %w", err)
	}
	if manifest.EncryptionType != "" && !strings.EqualFold(manifest.EncryptionType, "NONE") {
		return nil, fmt.Errorf("encrypted stream (%s) not supported", manifest.EncryptionType)
	}
	if len(manifest.URLs) == 0 {
		return nil, fmt.Errorf("manifest has no URLs")
	}
	return &StreamInfo{
		URL:      manifest.URLs[0],
		Quality:  parsed.Data.AudioQuality,
		MimeType: manifest.MimeType,
		Codec:    manifest.Codecs,
	}, nil
}

type playbackResponse struct {
	Data struct {
		AudioQuality     string `json:"audioQuality"`
		ManifestMimeType string `json:"manifestMimeType"`
		Manifest         string `json:"manifest"`
	} `json:"data"`
}

type manifestContent struct {
	MimeType       string   `json:"mimeType"`
	Codecs         string   `json:"codecs"`
	EncryptionType string   `json:"encryptionType"`
	URLs           []string `json:"urls"`
}

// fetchJSON GETs u, validates status, and detects FastAPI-style
// `{"detail":"..."}` upstream errors that come back with HTTP 200 (common when
// a monochrome mirror's TIDAL account is banned — it still serves 200 but
// returns no data). Returns the raw JSON body for caller-side decoding.
func (c *Client) fetchJSON(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, truncate(body, 200))
	}
	if detail := upstreamDetail(body); detail != "" {
		return nil, fmt.Errorf("upstream error: %s", detail)
	}
	return body, nil
}

// upstreamDetail returns the `detail` field from a FastAPI-style error body, or
// "" if the response contains real `data`.
func upstreamDetail(body []byte) string {
	var probe struct {
		Detail string          `json:"detail"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		return ""
	}
	if probe.Detail != "" && len(probe.Data) == 0 {
		return probe.Detail
	}
	return ""
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}

// tryAllHosts runs fn against each base in turn and returns the first success.
// If every host fails, a single error aggregating them all is returned.
func tryAllHosts[T any](bases []string, fn func(base string) (T, error)) (T, error) {
	var zero T
	if len(bases) == 0 {
		return zero, fmt.Errorf("no monochrome backends configured")
	}
	errs := make([]string, 0, len(bases))
	for _, base := range bases {
		result, err := fn(base)
		if err == nil {
			return result, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", base, err))
	}
	return zero, fmt.Errorf("all backends failed: %s", strings.Join(errs, "; "))
}

// Download streams streamURL to destPath. On any error the partial file is removed.
func (c *Client) Download(ctx context.Context, streamURL, destPath string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("stream request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stream download failed: status %d", resp.StatusCode)
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		os.Remove(destPath)
		return fmt.Errorf("copy stream: %w", copyErr)
	}
	if closeErr != nil {
		os.Remove(destPath)
		return fmt.Errorf("close dest file: %w", closeErr)
	}
	if n == 0 {
		os.Remove(destPath)
		return fmt.Errorf("empty stream body")
	}
	return nil
}
