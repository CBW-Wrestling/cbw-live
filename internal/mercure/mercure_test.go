package mercure

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTokenIsValidHS256(t *testing.T) {
	tok := Token("segredo", []string{"*"}, nil)
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT com %d partes", len(parts))
	}
	mac := hmac.New(sha256.New, []byte("segredo"))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) != parts[2] {
		t.Fatal("assinatura inválida")
	}
	payload, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims struct {
		Mercure struct{ Publish []string } `json:"mercure"`
	}
	json.Unmarshal(payload, &claims)
	if len(claims.Mercure.Publish) != 1 || claims.Mercure.Publish[0] != "*" {
		t.Fatalf("claims inesperadas: %s", payload)
	}
}

func TestReadSSE(t *testing.T) {
	stream := ": heartbeat\n\nid: a1\ndata: {\"x\":1}\n\nid: a2\nevent: msg\ndata: linha1\ndata: linha2\n\n"
	var got []Event
	last := readSSE(strings.NewReader(stream), "", func(e Event) { got = append(got, e) })
	if len(got) != 2 || last != "a2" {
		t.Fatalf("eventos=%d last=%q", len(got), last)
	}
	if string(got[1].Data) != "linha1\nlinha2" || got[1].Type != "msg" {
		t.Fatalf("evento 2 errado: %+v", got[1])
	}
}

func TestPublishAndSubscribe(t *testing.T) {
	received := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.Method {
		case http.MethodPost:
			r.ParseForm()
			received <- r.Form.Get("topic") + "|" + r.Form.Get("data")
			fmt.Fprint(w, "urn:uuid:1")
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "id: 1\ndata: {\"ok\":true}\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		}
	}))
	defer srv.Close()

	hub := NewHub(srv.URL, "k")
	if err := hub.Publish(context.Background(), "cbw/live/e/A", []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if got := <-received; got != `cbw/live/e/A|{"a":1}` {
		t.Fatalf("publish recebido: %s", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ev := make(chan Event, 1)
	go hub.Subscribe(ctx, []string{"t"}, func(e Event) { ev <- e; cancel() })
	select {
	case e := <-ev:
		if string(e.Data) != `{"ok":true}` {
			t.Fatalf("data: %s", e.Data)
		}
	case <-ctx.Done():
		t.Fatal("nenhum evento recebido")
	}
}
