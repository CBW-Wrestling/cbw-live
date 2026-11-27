package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"cbwlive/assets"
)

func get(h http.Handler, p string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
	return rec
}

func TestAssetsLogoFallsBackToPlaceholder(t *testing.T) {
	emb := fstest.MapFS{"logo.placeholder.svg": {Data: []byte("<svg>placeholder</svg>")}}
	h := assetsHandler(t.TempDir(), emb)
	rec := get(h, "/logo.svg")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "placeholder") {
		t.Fatalf("code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAssetsRealLogoWins(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "logo.svg"), []byte("<svg>real</svg>"), 0o644)
	emb := fstest.MapFS{"logo.placeholder.svg": {Data: []byte("<svg>placeholder</svg>")}}
	rec := get(assetsHandler(dir, emb), "/logo.svg")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "real") {
		t.Fatalf("code=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestAssetsEmbeddedAndMissing(t *testing.T) {
	emb := fstest.MapFS{"wizd-digital.svg": {Data: []byte("<svg>wizd</svg>")}}
	h := assetsHandler(t.TempDir(), emb)
	if rec := get(h, "/wizd-digital.svg"); rec.Code != 200 {
		t.Fatalf("embutido: code=%d", rec.Code)
	}
	if rec := get(h, "/nao-existe.svg"); rec.Code != 404 {
		t.Fatalf("ausente: code=%d", rec.Code)
	}
	if rec := get(h, "/../go.mod"); rec.Code == 200 {
		t.Fatal("path traversal servido")
	}
}

func TestFlagsWorkWithoutArena(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("GET /flags/{code}", newFlagProxy("http://127.0.0.1:1", t.TempDir(), assets.Files))
	for _, c := range []string{"sp", "BRA", "iri.svg"} {
		if rec := get(mux, "/flags/"+c); rec.Code != 200 {
			t.Errorf("%s: code=%d", c, rec.Code)
		}
	}
}

// Estado e país conhecidos usam a nossa bandeira; só código desconhecido vem do Arena.
func TestFlagsOnlyUnknownCodeComesFromArena(t *testing.T) {
	arena := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("<svg>arena</svg>"))
	}))
	defer arena.Close()
	mux := http.NewServeMux()
	mux.Handle("GET /flags/{code}", newFlagProxy(arena.URL, t.TempDir(), assets.Files))
	for code, fromArena := range map[string]bool{"sp": false, "bra": false, "clube-x": true} {
		rec := get(mux, "/flags/"+code)
		if got := strings.Contains(rec.Body.String(), ">arena<"); rec.Code != 200 || got != fromArena {
			t.Errorf("%s: code=%d do arena=%v", code, rec.Code, got)
		}
	}
}
