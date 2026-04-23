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
	"sync/atomic"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.Handler) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient([]string{srv.URL}, 5*time.Second), srv
}

func TestSearch_ParsesCanonicalResponse(t *testing.T) {
	// Shape modeled after a real response from api.monochrome.tf (fields
	// abbreviated). Both "artist" (singular) and "artists" (plural) are
	// returned upstream; we rely on the plural form.
	const canonicalBody = `{
		"version": "2.5",
		"data": {
			"limit": 25,
			"offset": 0,
			"totalNumberOfItems": 1,
			"items": [{
				"id": 36737274,
				"title": "Bohemian Rhapsody",
				"duration": 354,
				"isrc": "GBUM71029604",
				"audioQuality": "LOSSLESS",
				"artist": {"id": 8992, "name": "Queen"},
				"artists": [{"id": 8992, "name": "Queen", "type": "MAIN"}],
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

	matches, err := c.Search(context.Background(), "Bohemian Rhapsody Queen", 25)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if gotPath != "/search/" {
		t.Errorf("path: got %q, want /search/", gotPath)
	}
	if !strings.Contains(gotQuery, "s=") {
		t.Errorf("query must use s= param: got %q", gotQuery)
	}
	if strings.Contains(gotQuery, "i=") {
		t.Errorf("query must NOT use i= param (rejected by upstream): got %q", gotQuery)
	}
	if len(matches) != 1 {
		t.Fatalf("matches: got %d, want 1", len(matches))
	}
	m := matches[0]
	if m.TidalID != 36737274 || m.ISRC != "GBUM71029604" || m.Title != "Bohemian Rhapsody" {
		t.Errorf("match fields wrong: %+v", m)
	}
	if m.DurationSec != 354 || m.Quality != "LOSSLESS" {
		t.Errorf("match duration/quality wrong: %+v", m)
	}
	if len(m.Artists) != 1 || m.Artists[0] != "Queen" {
		t.Errorf("artists wrong: %+v", m.Artists)
	}
	if m.Album != "A Night at the Opera" {
		t.Errorf("album wrong: %q", m.Album)
	}
}

func TestSearch_EmptyQueryRejected(t *testing.T) {
	c := NewClient([]string{"http://example.invalid"}, time.Second)
	if _, err := c.Search(context.Background(), "", 10); err == nil {
		t.Fatal("expected error for empty query")
	}
	if _, err := c.Search(context.Background(), "   ", 10); err == nil {
		t.Fatal("expected error for whitespace-only query")
	}
}

func TestSearch_NoMatches(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":{"items":[]}}`)
	})
	c, _ := newTestClient(t, handler)
	matches, err := c.Search(context.Background(), "zzznonexistentxxx", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(matches))
	}
}

func TestSearch_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"detail":"Service Unavailable"}`, http.StatusServiceUnavailable)
	})
	c, _ := newTestClient(t, handler)
	if _, err := c.Search(context.Background(), "anything", 10); err == nil {
		t.Fatal("expected error on 503")
	}
}

func TestSearch_UpstreamAPIErrorTriggersFailover(t *testing.T) {
	// Monochrome mirrors return HTTP 200 with `{"detail":"Upstream API error"}`
	// when the backend's TIDAL account is banned. Our client must detect this
	// and fail over to the next configured host — which is the whole point of
	// multi-backend support.
	brokenHits := int32(0)
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&brokenHits, 1)
		fmt.Fprint(w, `{"detail":"Upstream API error"}`)
	}))
	t.Cleanup(broken.Close)

	healthyHits := int32(0)
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&healthyHits, 1)
		fmt.Fprint(w, `{"data":{"items":[{"id":1,"title":"ok","duration":30,"isrc":"X","audioQuality":"LOSSLESS","artists":[{"name":"A"}],"album":{"title":"B"}}]}}`)
	}))
	t.Cleanup(healthy.Close)

	c := NewClient([]string{broken.URL, healthy.URL}, 5*time.Second)
	matches, err := c.Search(context.Background(), "anything", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(matches) != 1 || matches[0].Title != "ok" {
		t.Fatalf("expected 1 match from healthy host, got %+v", matches)
	}
	if atomic.LoadInt32(&brokenHits) != 1 || atomic.LoadInt32(&healthyHits) != 1 {
		t.Errorf("expected both hosts hit once: broken=%d healthy=%d",
			atomic.LoadInt32(&brokenHits), atomic.LoadInt32(&healthyHits))
	}
}

func TestSearch_AllHostsFailAggregates(t *testing.T) {
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"detail":"Upstream API error"}`)
	}))
	t.Cleanup(a.Close)
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(b.Close)

	c := NewClient([]string{a.URL, b.URL}, 5*time.Second)
	_, err := c.Search(context.Background(), "anything", 10)
	if err == nil {
		t.Fatal("expected aggregated error when all hosts fail")
	}
	if !strings.Contains(err.Error(), "all backends failed") {
		t.Errorf("error should mention 'all backends failed': %v", err)
	}
}

func TestNewClient_TrimsAndFiltersBaseURLs(t *testing.T) {
	c := NewClient([]string{"  https://a.example.com/  ", "", "https://b.example.com"}, time.Second)
	hosts := c.Hosts()
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts, got %d: %v", len(hosts), hosts)
	}
	if hosts[0] != "https://a.example.com" || hosts[1] != "https://b.example.com" {
		t.Errorf("hosts wrong: %+v", hosts)
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

func TestGetStreamInfo_FailsOverBetweenHosts(t *testing.T) {
	// First host: upstream error. Second host: valid manifest. Expect success
	// and the second URL to be returned.
	broken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"detail":"Upstream API error"}`)
	}))
	t.Cleanup(broken.Close)

	manifestJSON := `{"mimeType":"audio/flac","codecs":"flac","encryptionType":"NONE","urls":["https://cdn.example.com/good.flac"]}`
	healthy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"data":{"audioQuality":"LOSSLESS","manifestMimeType":"application/vnd.tidal.bts","manifest":%q}}`,
			base64.StdEncoding.EncodeToString([]byte(manifestJSON)))
	}))
	t.Cleanup(healthy.Close)

	c := NewClient([]string{broken.URL, healthy.URL}, 5*time.Second)
	info, err := c.GetStreamInfo(context.Background(), 42, QualityLossless)
	if err != nil {
		t.Fatalf("GetStreamInfo: %v", err)
	}
	if info.URL != "https://cdn.example.com/good.flac" {
		t.Errorf("expected URL from healthy host, got %q", info.URL)
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
	c := NewClient([]string{"http://example.invalid"}, time.Second)
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
