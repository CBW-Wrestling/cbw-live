// Package webhook recebe os webhooks do Arena (HTTP POST), registra o payload
// bruto e dispara uma sincronização.
package webhook

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"cbwlive/internal/live"
)

type Handler struct {
	Token   string // token configurado no webhook do Arena
	Enforce bool   // rejeitar quando o token não confere
	Log     *live.Recorder
	// OnEvent recebe a notificação do Arena: {"action":"insert","entity":"RefereeChallenge","id":"…"}
	OnEvent func(entity, action, id string)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "erro lendo corpo", http.StatusBadRequest)
		return
	}

	tokenOK := h.Token == "" || tokenMatches(body, h.Token)
	h.Log.Write(map[string]any{
		"contentType": r.Header.Get("Content-Type"),
		"userAgent":   r.Header.Get("User-Agent"),
		"tokenOK":     tokenOK,
		"body":        live.RawOrString(body),
	})
	if !tokenOK && h.Enforce {
		http.Error(w, "token inválido", http.StatusUnauthorized)
		return
	}
	if !tokenOK {
		slog.Warn("webhook: token não encontrado/conferido no payload (modo descoberta, aceito mesmo assim)")
	}
	slog.Info("webhook recebido", "bytes", len(body))
	if h.OnEvent != nil {
		var n struct{ Entity, Action, ID string }
		json.Unmarshal(body, &n)
		h.OnEvent(n.Entity, n.Action, n.ID)
	}
	w.WriteHeader(http.StatusNoContent)
}

// tokenMatches procura a chave "token" no primeiro nível do JSON.
// O nome exato do campo será confirmado quando virmos o primeiro payload real.
func tokenMatches(body []byte, want string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(body, &m) != nil {
		return false
	}
	var got string
	if json.Unmarshal(m["token"], &got) != nil {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
