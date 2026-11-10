// Package config carrega a configuração do .env e das variáveis de ambiente
// (variáveis de ambiente têm precedência sobre o arquivo).
package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ArenaURL          string
	ArenaClientID     string
	ArenaClientSecret string
	ArenaAPIKey       string
	ArenaEventID      string

	MercureHubURL string
	MercureJWTKey string

	ListenAddr          string
	DataDir             string
	AssetsDir           string
	WebhookToken        string
	WebhookTokenEnforce bool
	Resync              time.Duration
}

func Load(path string) (*Config, error) {
	file := map[string]string{}
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			file[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	get := func(k, def string) string {
		if v, ok := os.LookupEnv(k); ok {
			return v
		}
		if v, ok := file[k]; ok && v != "" {
			return v
		}
		return def
	}

	resync, err := strconv.Atoi(get("RESYNC_SECONDS", "15"))
	if err != nil {
		return nil, fmt.Errorf("RESYNC_SECONDS inválido: %w", err)
	}
	c := &Config{
		ArenaURL:            strings.TrimRight(get("ARENA_URL", "http://localhost:8080"), "/"),
		ArenaClientID:       get("ARENA_CLIENT_ID", ""),
		ArenaClientSecret:   get("ARENA_CLIENT_SECRET", ""),
		ArenaAPIKey:         get("ARENA_API_KEY", ""),
		ArenaEventID:        get("ARENA_EVENT_ID", ""),
		MercureHubURL:       get("MERCURE_HUB_URL", "http://localhost:8080/.well-known/mercure"),
		MercureJWTKey:       get("MERCURE_JWT_KEY", ""),
		ListenAddr:          get("LISTEN_ADDR", ":8095"),
		DataDir:             get("DATA_DIR", "data"),
		AssetsDir:           get("ASSETS_DIR", "assets"),
		WebhookToken:        get("WEBHOOK_TOKEN", ""),
		WebhookTokenEnforce: get("WEBHOOK_TOKEN_ENFORCE", "false") == "true",
		Resync:              time.Duration(resync) * time.Second,
	}

	var missing []string
	for k, v := range map[string]string{
		"ARENA_CLIENT_ID": c.ArenaClientID, "ARENA_CLIENT_SECRET": c.ArenaClientSecret,
		"ARENA_API_KEY": c.ArenaAPIKey, "MERCURE_JWT_KEY": c.MercureJWTKey,
	} {
		if v == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("configuração incompleta, faltam: %s (rode ./scripts/setup-env.sh)", strings.Join(missing, ", "))
	}
	return c, nil
}
