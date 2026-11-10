// Package arena é o cliente da API REST do UWW Arena (OAuth2 com o grant
// customizado "api_key", usado pelo placar oficial).
package arena

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const apiKeyGrant = "https://arena.uww.io/grants/api_key"

type Client struct {
	base, clientID, clientSecret, apiKey string
	http                                 *http.Client

	mu      sync.Mutex
	token   string
	expires time.Time
}

func New(base, clientID, clientSecret, apiKey string) *Client {
	return &Client{
		base: base, clientID: clientID, clientSecret: clientSecret, apiKey: apiKey,
		http: &http.Client{
			Timeout:   5 * time.Second,
			Transport: &http.Transport{MaxIdleConnsPerHost: 32, IdleConnTimeout: 90 * time.Second},
		},
	}
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	form := url.Values{
		"grant_type": {apiKeyGrant}, "client_id": {c.clientID},
		"client_secret": {c.clientSecret}, "api_key": {c.apiKey},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/oauth/v2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("arena token: %w", err)
	}
	defer resp.Body.Close()
	var t struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
		Description string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return "", fmt.Errorf("arena token: resposta inválida (%d): %w", resp.StatusCode, err)
	}
	if t.AccessToken == "" {
		return "", fmt.Errorf("arena token: %s: %s", t.Error, t.Description)
	}
	ttl := time.Duration(t.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = 30 * time.Minute
	}
	c.token, c.expires = t.AccessToken, time.Now().Add(ttl-30*time.Second)
	return c.token, nil
}

func (c *Client) invalidate() {
	c.mu.Lock()
	c.token = ""
	c.mu.Unlock()
}

// GetRaw faz GET em /api/json/{path} e devolve o corpo bruto.
// Em 401 renova o token e tenta mais uma vez.
func (c *Client) GetRaw(ctx context.Context, path string) ([]byte, error) {
	for attempt := 0; attempt < 2; attempt++ {
		tok, err := c.accessToken(ctx)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/json/"+strings.TrimLeft(path, "/"), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("Accept", "application/json")
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized && attempt == 0 {
			c.invalidate()
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("arena GET %s: HTTP %d: %.200s", path, resp.StatusCode, body)
		}
		return body, nil
	}
	return nil, fmt.Errorf("arena GET %s: não autorizado", path)
}

func (c *Client) Get(ctx context.Context, path string, out any) error {
	body, err := c.GetRaw(ctx, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

// ---- Tipos usados pelo sincronizador ----

type Mat struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SportEvent struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	FullName          string `json:"fullName"`
	AddressLocality   string `json:"addressLocality"`
	Athlete1Color     string `json:"athlete1Color"`
	Athlete1TextColor string `json:"athlete1TextColor"`
	Athlete2Color     string `json:"athlete2Color"`
	Athlete2TextColor string `json:"athlete2TextColor"`
	Mats              []Mat  `json:"mats"`
}

func (c *Client) ScoreboardEvents(ctx context.Context) ([]SportEvent, error) {
	var r struct {
		Events []SportEvent `json:"events"`
	}
	err := c.Get(ctx, "scoreboard/sport-event", &r)
	return r.Events, err
}

// OpenFights devolve as lutas abertas do tapete, sem interpretar o formato
// (ainda estamos descobrindo o payload).
func (c *Client) OpenFights(ctx context.Context, matID string) ([]json.RawMessage, error) {
	var r struct {
		Fights []json.RawMessage `json:"fights"`
	}
	err := c.Get(ctx, "scoreboard/fight/open/"+url.PathEscape(matID), &r)
	return r.Fights, err
}
