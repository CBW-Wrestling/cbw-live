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
	// Scoreboard: modelo visual do placar (ver ScoreboardModels).
	Scoreboard string `json:"scoreboard"`
	// Position: onde o placar fica na tela (ver Positions).
	Position string `json:"position"`
	// IntroPosition: onde a apresentação fica: "auto" (a padrão do modelo) ou uma de Positions.
	IntroPosition string `json:"introPosition"`
	// IntroStyle: modelo da apresentação da luta (ver IntroStyles).
	IntroStyle string `json:"introStyle"`
	// Anim: animação de entrada/saída: "auto" (a do modelo), "fade", "slide", "drop", "zoom" ou "wipe".
	Anim string `json:"anim"`
}

// Modelos disponíveis; precisam existir em web/themes.js.
var (
	ScoreboardModels = []string{"classic", "broadcast", "stack", "compact", "league", "futy", "arena", "clean", "duel", "flo", "dual", "judo", "uww"}
	IntroStyles      = []string{"lower", "versus", "card", "topbar", "arena", "clean", "split", "ticker", "uww"}
	Positions        = []string{"top-start", "top-center", "top-end", "bottom-start", "bottom-center", "bottom-end"}
	Animations       = []string{"auto", "fade", "slide", "drop", "zoom", "wipe"}
)

func oneOf(v string, list []string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func DefaultSettings() Settings {
	return Settings{CollapseMode: "auto", AfterScoreSeconds: 6, ShowStartIn: false, ShowEventInfo: true, Background: "transparent", AutoIntro: true,
		Scoreboard: "classic", Position: "top-start", IntroPosition: "auto", IntroStyle: "lower", Anim: "auto"}
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
	if !oneOf(s.Scoreboard, ScoreboardModels) {
		return fmt.Errorf("scoreboard deve ser um de %v", ScoreboardModels)
	}
	if !oneOf(s.Position, Positions) {
		return fmt.Errorf("position deve ser um de %v", Positions)
	}
	if s.IntroPosition != "auto" && !oneOf(s.IntroPosition, Positions) {
		return fmt.Errorf(`introPosition deve ser "auto" ou um de %v`, Positions)
	}
	if !oneOf(s.IntroStyle, IntroStyles) {
		return fmt.Errorf("introStyle deve ser um de %v", IntroStyles)
	}
	if !oneOf(s.Anim, Animations) {
		return fmt.Errorf("anim deve ser um de %v", Animations)
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
	// valores antigos ("left"/"center") viraram as posições nomeadas
	switch loaded.Position {
	case "left":
		loaded.Position = "top-start"
	case "center":
		loaded.Position = "top-center"
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
