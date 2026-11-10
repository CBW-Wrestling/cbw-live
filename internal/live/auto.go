package live

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"time"
)

// Automações a partir dos dados reais do UWW Timing / Arena (capturados em 09/11/2026):
//
//   - Apresentação: Timing publica state="starting" ao carregar a luta e "running" ao iniciar o cronômetro.
//   - Vitória: mensagem ao vivo traz message.detail {winner, victory, victoryTime}.
//   - Challenge: webhook RefereeChallenge (insert/update) → /api/json/challenge/get/{id}
//     traz fighterId (quem pediu) e challengeStatus.
//
// Atendimento (sangue/lesão) e passividade não aparecem nos dados — continuam manuais.

// ChallengeHold é quanto tempo o resultado do challenge fica no ar.
const ChallengeHold = 5 * time.Second

func setControl(m *MatState, req ControlRequest, now time.Time) {
	cs := controlState{ControlRequest: req, UpdatedAt: now.UnixMilli(), StartedAt: now.UnixMilli(), Auto: true}
	b, _ := json.Marshal(cs)
	m.Control[req.Graphic] = b
}

func controlOf(m *MatState, graphic string) (controlState, bool) {
	raw, ok := m.Control[graphic]
	if !ok {
		return controlState{}, false
	}
	var cs controlState
	return cs, json.Unmarshal(raw, &cs) == nil
}

// victoryName traduz o identificador ("VFA") para o nome oficial do Arena ("VICTORY BY FALL").
func victoryName(fight json.RawMessage, id string) string {
	var f struct {
		VictoryTypes []struct {
			Identifier string `json:"identifier"`
			Name       string `json:"name"`
		} `json:"victoryTypesArray"`
	}
	if json.Unmarshal(fight, &f) == nil {
		for _, v := range f.VictoryTypes {
			if v.Identifier == id {
				return v.Name
			}
		}
	}
	return id
}

// autoFromLive aplica as automações de uma mensagem ao vivo. Roda dentro de Store.Update.
// Devolve true se algum controle mudou.
func (s *Syncer) autoFromLive(m *MatState, lm liveMessage, now time.Time) bool {
	if lm.Source == "timing" {
		return false // eventos diretos do Timing (pre/rdy/wmh) já aplicaram os gráficos
	}
	changed := false
	msg := lm.Message

	// Apresentação automática
	autoIntro := s.Settings == nil || s.Settings.Get().AutoIntro
	intro, hasIntro := controlOf(m, "intro")
	switch {
	case autoIntro && lm.State == "starting" && msg.ID != "" && m.introFor != msg.ID:
		m.introFor = msg.ID
		setControl(m, ControlRequest{Graphic: "intro", Visible: true}, now)
		changed = true
	case lm.State == "running" && hasIntro && intro.Auto:
		delete(m.Control, "intro")
		changed = true
	}

	// Vitória
	vic, hasVic := controlOf(m, "victory")
	if d := msg.Detail; d != nil && d.Victory != "" && d.Winner != "" {
		side := ""
		switch d.Winner {
		case msg.Home.Code:
			side = "home"
		case msg.Away.Code:
			side = "away"
		}
		name := victoryName(m.Fight, d.Victory)
		if side != "" && (!hasVic || vic.Side != side || vic.VictoryType != name) {
			setControl(m, ControlRequest{Graphic: "victory", Visible: true, Side: side, VictoryType: name}, now)
			changed = true
		}
	} else if hasVic && vic.Auto && (msg.Detail == nil || msg.Detail.Winner == "") {
		// resultado desfeito na mesa
		delete(m.Control, "victory")
		changed = true
	}
	return changed
}

// HandleEntity reage a um webhook do Arena (entity/action/id).
func (s *Syncer) HandleEntity(ctx context.Context, entity, action, id string) {
	if entity != "RefereeChallenge" || (action != "insert" && action != "update") || id == "" {
		return
	}
	var r struct {
		Challenge struct {
			FightID   string `json:"fightId"`
			FighterID string `json:"fighterId"`
			Status    *bool  `json:"challengeStatus"`
			Time      string `json:"challengeTime"`
		} `json:"challenge"`
	}
	if err := s.Arena.Get(ctx, "challenge/get/"+url.PathEscape(id), &r); err != nil {
		slog.Warn("challenge: falha ao consultar o Arena", "id", id, "err", err)
		return
	}
	ch := r.Challenge
	if s.timingActiveForFight(ch.FightID) {
		return // o Timing conectado direto já mostrou cada etapa do challenge
	}
	status := "challenge"
	if ch.Status != nil {
		status = map[bool]string{true: "won", false: "lost"}[*ch.Status]
	}

	for _, mat := range s.Store.All() {
		var f struct {
			ID         string `json:"id"`
			Fighter1ID string `json:"fighter1Id"`
			Fighter2ID string `json:"fighter2Id"`
		}
		json.Unmarshal(mat.Fight, &f)
		if f.ID != ch.FightID && mat.LiveFightID != ch.FightID {
			continue
		}
		side := "away"
		if ch.FighterID == f.Fighter1ID {
			side = "home"
		}
		now := time.Now()
		key := Key(mat.EventID, mat.MatName)
		state, _ := s.Store.Update(key, func(m *MatState) bool {
			setControl(m, ControlRequest{Graphic: "challenge", Visible: true, Side: side, Status: status}, now)
			return true
		})
		s.publish(ctx, mat.EventID, mat.MatName, message{Type: "control", State: &state})
		slog.Info("challenge automático", "mat", mat.MatName, "lado", side, "status", status, "tempo", ch.Time)

		// sai sozinho depois de ChallengeHold, se ninguém mexeu nele nesse meio-tempo
		stamp := now.UnixMilli()
		eventID, matName := mat.EventID, mat.MatName
		time.AfterFunc(ChallengeHold, func() {
			st, changed := s.Store.Update(key, func(m *MatState) bool {
				cs, ok := controlOf(m, "challenge")
				if !ok || !cs.Auto || cs.UpdatedAt != stamp {
					return false
				}
				delete(m.Control, "challenge")
				return true
			})
			if changed {
				s.publish(context.Background(), eventID, matName, message{Type: "control", State: &st})
			}
		})
		return
	}
	slog.Warn("challenge: luta não está em nenhum tapete acompanhado", "fight", ch.FightID)
}
