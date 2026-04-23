package monochrome

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, 5*time.Second), srv
}

func TestSearchByISRC_ParsesCanonicalResponse(t *testing.T) {
	const canonicalBody = `{
		"version": "2.10",
		"data": {
			"limit": 5,
			"offset": 0,
			"totalNumberOfItems": 1,
			"items": [{
				"id": 77640691,
				"title": "Bohemian Rhapsody",
				"duration": 354,
				"isrc": "GBUM71029600",
				"audioQuality": "HI_RES_LOSSLESS",
				"artists": [{"id": 1, "name": "Queen"}],
				"album": {"id": 2, "title": "A Night at the Opera"}
			}]
		}
	}`

	var gotPath, gotQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, canonicalBody)
	})
	c, _ := newTestClient(t, handler)

	matches, err := c.SearchByISRC(context.Background(), "GBUM71029600")
	if err != nil {
		t.Fatalf("SearchByISRC: %v", err)
	}
	if gotPath != "/search/" {
		t.Errorf("path: got %q, want /search/", gotPath)
	}
	if !strings.Contains(gotQuery, "i=GBUM71029600") {
		t.Errorf("query must contain ISRC param: got %q", gotQuery)
	}
	if len(matches) != 1 {
		t.Fatalf("matches: got %d, want 1", len(matches))
	}
	m := matches[0]
	if m.TidalID != 77640691 || m.ISRC != "GBUM71029600" || m.Title != "Bohemian Rhapsody" {
		t.Errorf("match fields wrong: %+v", m)
	}
	if m.DurationSec != 354 || m.Quality != "HI_RES_LOSSLESS" {
		t.Errorf("match duration/quality wrong: %+v", m)
	}
	if len(m.Artists) != 1 || m.Artists[0] != "Queen" {
		t.Errorf("artists wrong: %+v", m.Artists)
	}
	if m.Album != "A Night at the Opera" {
		t.Errorf("album wrong: %q", m.Album)
	}
}

func TestSearchByISRC_EmptyISRCRejected(t *testing.T) {
	c := NewClient("http://example.invalid", time.Second)
	if _, err := c.SearchByISRC(context.Background(), ""); err == nil {
		t.Fatal("expected error for empty ISRC")
	}
	if _, err := c.SearchByISRC(context.Background(), "   "); err == nil {
		t.Fatal("expected error for whitespace-only ISRC")
	}
}

func TestSearchByISRC_NoMatches(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"items":[]}}`)
	})
	c, _ := newTestClient(t, handler)
	matches, err := c.SearchByISRC(context.Background(), "ZZZZ00000000")
	if err != nil {
		t.Fatalf("SearchByISRC: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestSearchByISRC_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"Service Unavailable"}`, http.StatusServiceUnavailable)
	})
	c, _ := newTestClient(t, handler)
	if _, err := c.SearchByISRC(context.Background(), "USRC11300135"); err == nil {
		t.Fatal("expected error on 503")
	}
}

func TestGetStreamInfo_DecodesBase64Manifest(t *testing.T) {
	manifestJSON := `{"mimeType":"audio/flac","codecs":"flac","encryptionType":"NONE","urls":["https://cdn.example.com/stream.flac?sig=abc"]}`
	body := fmt.Sprintf(`{
		"data": {
			"trackId": 77640691,
			"audioQuality": "HI_RES_LOSSLESS",
			"bitDepth": 24,
			"sampleRate": 96000,
			"manifestMimeType": "application/vnd.tidal.bts",
			"manifest": %q
		}
	}`, base64.StdEncoding.EncodeToString([]byte(manifestJSON)))

	var gotQuery string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, body)
	})
	c, _ := newTestClient(t, handler)

	info, err := c.GetStreamInfo(context.Background(), 77640691, QualityHiRes)
	if err != nil {
		t.Fatalf("GetStreamInfo: %v", err)
	}
	if info.URL != "https://cdn.example.com/stream.flac?sig=abc" {
		t.Errorf("URL wrong: %q", info.URL)
	}
	if info.Quality != "HI_RES_LOSSLESS" || info.Codec != "flac" || info.MimeType != "audio/flac" {
		t.Errorf("info fields wrong: %+v", info)
	}
	if !strings.Contains(gotQuery, "id=77640691") || !strings.Contains(gotQuery, "quality=HI_RES_LOSSLESS") {
		t.Errorf("query wrong: %q", gotQuery)
	}
}

func TestGetStreamInfo_RejectsDASHManifest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"manifestMimeType":"application/dash+xml","manifest":"PHhtbD4="}}`)
	})
	c, _ := newTestClient(t, handler)
	_, err := c.GetStreamInfo(context.Background(), 1, QualityLossless)
	if err == nil {
		t.Fatal("expected error on DASH manifest")
	}
	if !strings.Contains(err.Error(), "unsupported manifest") {
		t.Errorf("error message should mention unsupported manifest: %v", err)
	}
}

func TestGetStreamInfo_RejectsEncryptedStream(t *testing.T) {
	manifestJSON := `{"mimeType":"audio/mp4","codecs":"mp4a.40.2","encryptionType":"OLD_AES","urls":["x"]}`
	body := fmt.Sprintf(`{"data":{"manifestMimeType":"application/vnd.tidal.bts","manifest":%q}}`,
		base64.StdEncoding.EncodeToString([]byte(manifestJSON)))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	})
	c, _ := newTestClient(t, handler)
	_, err := c.GetStreamInfo(context.Background(), 1, QualityLossless)
	if err == nil || !strings.Contains(err.Error(), "encrypted") {
		t.Fatalf("expected encryption error, got %v", err)
	}
}

func TestGetStreamInfo_RejectsInvalidID(t *testing.T) {
	c := NewClient("http://example.invalid", time.Second)
	if _, err := c.GetStreamInfo(context.Background(), 0, QualityLossless); err == nil {
		t.Fatal("expected error on id=0")
	}
	if _, err := c.GetStreamInfo(context.Background(), -5, QualityLossless); err == nil {
		t.Fatal("expected error on negative id")
	}
}

func TestDownload_StreamsToFile(t *testing.T) {
	const payload = "FLACFAKEBYTES12345"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/flac")
		fmt.Fprint(w, payload)
	})
	c, srv := newTestClient(t, handler)

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.flac")
	if err := c.Download(context.Background(), srv.URL+"/stream", dest); err != nil {
		t.Fatalf("Download: %v", err)
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != payload {
		t.Errorf("payload mismatch: got %q want %q", string(got), payload)
	}
}

func TestDownload_CleansPartialOnError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusGone)
	})
	c, srv := newTestClient(t, handler)

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.flac")
	if err := c.Download(context.Background(), srv.URL+"/stream", dest); err == nil {
		t.Fatal("expected error on 410")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("expected dest file to not exist after error, stat err: %v", err)
	}
}

func TestDownload_RejectsEmptyBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	c, srv := newTestClient(t, handler)

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.flac")
	if err := c.Download(context.Background(), srv.URL+"/stream", dest); err == nil {
		t.Fatal("expected error on empty body")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("expected dest file to not exist after empty body, stat err: %v", err)
	}
}
