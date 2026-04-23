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

// expected manifest type for plain lossless FLAC (TIDAL BTS container).
const manifestTypeBTS = "application/vnd.tidal.bts"

// Client talks to a monochrome.tf-compatible instance of the hifi-api.
// Zero value is not usable — use NewClient.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client pointing at baseURL (e.g. "https://api.monochrome.tf").
// timeout bounds individual HTTP requests; use >= 60s to accommodate the ~10MB
// CDN stream requests.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

// TrackMatch is a normalized catalog hit returned by Search*.
type TrackMatch struct {
	TidalID     int
	ISRC        string
	Title       string
	Artists     []string
	Album       string
	DurationSec int
	Quality     string // upstream audioQuality: HI_RES_LOSSLESS, LOSSLESS, HIGH, LOW
}

// StreamInfo describes a resolved, signed CDN URL for the actual audio bytes.
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
	return c.doSearch(ctx, url.Values{
		"s":     {query},
		"limit": {fmt.Sprintf("%d", limit)},
	})
}

func (c *Client) doSearch(ctx context.Context, q url.Values) ([]TrackMatch, error) {
	u := c.baseURL + "/search/?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("search failed: status %d: %s", resp.StatusCode, string(body))
	}
	var parsed searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
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
	u := fmt.Sprintf("%s/track/?id=%d&quality=%s", c.baseURL, tidalID, url.QueryEscape(quality))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("track request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("track fetch failed: status %d: %s", resp.StatusCode, string(body))
	}
	var parsed playbackResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
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
