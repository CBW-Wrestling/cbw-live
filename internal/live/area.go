package live

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
)

// AreaEvent é o "evento" reservado das URLs por área: /overlay.html?mat=A
// acompanha o tapete A em qualquer evento, sem trocar o link durante o dia.
const AreaEvent = "area"

// ActiveArea devolve o estado do evento que está valendo agora no tapete
// matName. Em empate mantém o escolhido antes.
func (s *Syncer) ActiveArea(matName string) (MatState, bool) {
	s.amu.Lock()
	defer s.amu.Unlock()
	return s.activeArea(matName)
}

// activeArea exige s.amu.
func (s *Syncer) activeArea(matName string) (MatState, bool) {
	var best MatState
	var bestRank [4]int64
	found := false
	for _, m := range s.Store.All() {
		if m.MatName != matName {
			continue
		}
		r := s.areaRank(m)
		cmp := slices.Compare(r[:], bestRank[:])
		if !found || cmp > 0 || (cmp == 0 && m.EventID == s.areas[matName]) {
			best, bestRank, found = m, r, true
		}
	}
	return best, found
}

// areaRank, em ordem de peso: evento ainda listado no Arena, UWW Timing
// conectado, última mensagem ao vivo, tem luta aberta. Exige s.amu.
func (s *Syncer) areaRank(m MatState) (r [4]int64) {
	if s.listed == nil || s.listed[Key(m.EventID, m.MatName)] {
		r[0] = 1
	}
	if m.TimingAddr != "" {
		r[1] = 1
	}
	if m.LiveAt != nil {
		r[2] = m.LiveAt.UnixNano()
	}
	if m.OpenFights > 0 {
		r[3] = 1
	}
	return r
}

// publishArea repete no tópico fixo da área o que é do evento ativo. Quando o
// evento ativo muda, manda o estado completo do novo antes.
// msg nil só reavalia qual é o evento ativo.
func (s *Syncer) publishArea(ctx context.Context, matName string, eventID string, msg []byte) {
	s.amu.Lock()
	defer s.amu.Unlock()
	st, ok := s.activeArea(matName)
	if !ok {
		return
	}
	topic := Topic(AreaEvent, matName)
	if st.EventID != s.areas[matName] {
		s.areas[matName] = st.EventID
		slog.Info("área: evento ativo", "mat", matName, "evento", st.EventName)
		msg, _ = json.Marshal(message{Type: "snapshot", State: &st})
	} else if msg == nil || eventID != st.EventID {
		return
	}
	if err := s.Hub.Publish(ctx, topic, msg); err != nil {
		slog.Warn("mercure: falha ao publicar", "area", matName, "err", err)
	}
}
