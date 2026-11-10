package hubproxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestProxy(t *testing.T) {
	var gotAuth, gotQuery string
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotQuery = r.Header.Get("Authorization"), r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
	}))
	defer hub.Close()

	h, err := New(hub.URL+"/.well-known/mercure", "SUBTOKEN")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		method, topic string
		want          int
	}{
		{http.MethodGet, "cbw/live/e/A", http.StatusOK},
		{http.MethodGet, "http://arena.uww.org/live/e/A", http.StatusForbidden},
		{http.MethodPost, "cbw/live/e/A", http.StatusMethodNotAllowed},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, "/hub?topic="+url.QueryEscape(c.topic), nil))
		if rec.Code != c.want {
			t.Errorf("%s %s: status %d, esperado %d", c.method, c.topic, rec.Code, c.want)
		}
	}
	if gotAuth != "Bearer SUBTOKEN" {
		t.Errorf("Authorization repassado: %q", gotAuth)
	}
	if gotQuery != "topic="+url.QueryEscape("cbw/live/e/A") {
		t.Errorf("query repassada: %q", gotQuery)
	}
}
