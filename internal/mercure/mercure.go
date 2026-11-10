// Package mercure publica e assina tópicos no hub Mercure do Arena.
// JWT HS256 gerado localmente (sem dependências externas).
package mercure

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Token gera um JWT HS256 com a claim "mercure".
func Token(key string, publish, subscribe []string) string {
	b64 := base64.RawURLEncoding.EncodeToString
	header := b64([]byte(`{"alg":"HS256","typ":"JWT"}`))
	claims := map[string]any{"mercure": map[string][]string{"publish": publish, "subscribe": subscribe}}
	payload, _ := json.Marshal(claims)
	unsigned := header + "." + b64(payload)
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(unsigned))
	return unsigned + "." + b64(mac.Sum(nil))
}

type Hub struct {
	URL       string
	pubToken  string
	subToken  string
	http      *http.Client
	streaming *http.Client
}

func NewHub(hubURL, key string) *Hub {
	return &Hub{
		URL:       hubURL,
		pubToken:  Token(key, []string{"*"}, nil),
		subToken:  Token(key, nil, []string{"*"}),
		http:      &http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: 32}},
		streaming: &http.Client{}, // sem timeout: conexão SSE de longa duração
	}
}

// SubscriberToken é usado pelo proxy dos overlays.
func (h *Hub) SubscriberToken() string { return h.subToken }

func (h *Hub) Publish(ctx context.Context, topic string, data []byte) error {
	form := url.Values{"topic": {topic}, "data": {string(data)}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.URL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+h.pubToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := h.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mercure publish %s: HTTP %d", topic, resp.StatusCode)
	}
	return nil
}

type Event struct {
	ID    string
	Type  string
	Topic string // não vem no SSE; preenchido quando o payload não traz
	Data  []byte
}

// Subscribe mantém uma assinatura SSE com reconexão automática e Last-Event-ID,
// chamando fn para cada mensagem. Bloqueia até ctx ser cancelado.
func (h *Hub) Subscribe(ctx context.Context, topics []string, fn func(Event)) {
	lastID := ""
	backoff := time.Second
	for ctx.Err() == nil {
		q := url.Values{}
		for _, t := range topics {
			q.Add("topic", t)
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, h.URL+"?"+q.Encode(), nil)
		req.Header.Set("Authorization", "Bearer "+h.subToken)
		req.Header.Set("Accept", "text/event-stream")
		if lastID != "" {
			req.Header.Set("Last-Event-ID", lastID)
		}
		resp, err := h.streaming.Do(req)
		if err == nil && resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("HTTP %d", resp.StatusCode)
			resp.Body.Close()
		}
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Warn("mercure: falha ao assinar, tentando de novo", "err", err, "em", backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			backoff = min(backoff*2, 15*time.Second)
			continue
		}
		slog.Info("mercure: assinatura ativa", "topics", topics)
		backoff = time.Second
		lastID = readSSE(resp.Body, lastID, fn)
		resp.Body.Close()
	}
}

func readSSE(r io.Reader, lastID string, fn func(Event)) string {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	var ev Event
	var data []string
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if len(data) > 0 {
				ev.Data = []byte(strings.Join(data, "\n"))
				if ev.ID != "" {
					lastID = ev.ID
				}
				fn(ev)
			}
			ev, data = Event{}, nil
		case strings.HasPrefix(line, ":"):
			// comentário / heartbeat
		default:
			k, v, _ := strings.Cut(line, ":")
			v = strings.TrimPrefix(v, " ")
			switch k {
			case "id":
				ev.ID = v
			case "event":
				ev.Type = v
			case "data":
				data = append(data, v)
			}
		}
	}
	return lastID
}
