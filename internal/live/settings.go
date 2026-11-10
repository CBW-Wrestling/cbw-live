package live

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"cbwlive/internal/mercure"
)

// SettingsTopic é o tópico Mercure em que os overlays recebem as configurações globais.
const SettingsTopic = "cbw/settings"

// Settings são preferências de exibição que valem para todos os tapetes.
type Settings struct {
	// CollapseMode: "auto" (recolhe com o cronômetro correndo), "never" (sempre expandido)
	// ou "always" (sempre recolhido).
	CollapseMode string `json:"collapseMode"`
	// AfterScoreSeconds: no modo automático, quanto tempo o placar fica expandido após um ponto.
	AfterScoreSeconds int `json:"afterScoreSeconds"`
	// ShowStartIn: mostra "Início em mm:ss" na apresentação da luta.
	ShowStartIn bool `json:"showStartIn"`
	// ShowEventInfo: mostra nome do evento e cidade na apresentação da luta.
	ShowEventInfo bool `json:"showEventInfo"`
	// Background: "transparent" (Browser Source com alfa), "green" (#00FF00) ou "blue" (#0000FF)
	// para chroma key. Com chroma, sombras e transparências do gráfico ficam sólidas.
	Background string `json:"background"`
	// AutoIntro: mostra a apresentação quando o Timing carrega a luta e tira quando o cronômetro inicia.
	AutoIntro bool `json:"autoIntro"`
}

func DefaultSettings() Settings {
	return Settings{CollapseMode: "auto", AfterScoreSeconds: 6, ShowStartIn: false, ShowEventInfo: true, Background: "transparent", AutoIntro: true}
}

func (s Settings) Validate() error {
	switch s.CollapseMode {
	case "auto", "never", "always":
	default:
		return fmt.Errorf(`collapseMode deve ser "auto", "never" ou "always"`)
	}
	switch s.Background {
	case "transparent", "green", "blue":
	default:
		return fmt.Errorf(`background deve ser "transparent", "green" ou "blue"`)
	}
	if s.AfterScoreSeconds < 0 || s.AfterScoreSeconds > 60 {
		return fmt.Errorf("afterScoreSeconds deve estar entre 0 e 60")
	}
	return nil
}

// SettingsStore guarda as configurações em memória e em disco (JSON).
type SettingsStore struct {
	mu   sync.RWMutex
	path string
	cur  Settings
	hub  *mercure.Hub
}

func NewSettingsStore(dir string, hub *mercure.Hub) (*SettingsStore, error) {
	st := &SettingsStore{path: filepath.Join(dir, "settings.json"), cur: DefaultSettings(), hub: hub}
	b, err := os.ReadFile(st.path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return st, nil
	case err != nil:
		return nil, err
	}
	loaded := DefaultSettings() // campos novos ganham o valor padrão
	if err := json.Unmarshal(b, &loaded); err != nil {
		return nil, fmt.Errorf("%s inválido: %w", st.path, err)
	}
	if err := loaded.Validate(); err != nil {
		slog.Warn("configurações inválidas no disco; usando padrão", "err", err)
		return st, nil
	}
	st.cur = loaded
	return st, nil
}

func (st *SettingsStore) Get() Settings {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.cur
}

// Set valida, grava em disco (escrita atômica) e publica para os overlays.
func (st *SettingsStore) Set(ctx context.Context, s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	tmp := st.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, st.path); err != nil {
		return err
	}
	st.mu.Lock()
	st.cur = s
	st.mu.Unlock()
	msg, _ := json.Marshal(map[string]any{"type": "settings", "settings": s})
	if err := st.hub.Publish(ctx, SettingsTopic, msg); err != nil {
		slog.Warn("mercure: falha ao publicar configurações", "err", err)
	}
	return nil
}
