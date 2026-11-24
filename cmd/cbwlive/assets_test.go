package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
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
