package live

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// Mensagens reais do UWW Timing gravadas em 09/11/2026 (testdata/timing_live.json):
// uma "starting" (luta carregada) e as do fim da luta com message.detail.
func loadTimingFixture(t *testing.T) []liveMessage {
	t.Helper()
	b, err := os.ReadFile("testdata/timing_live.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	out := make([]liveMessage, len(raw))
	for i, r := range raw {
		if err := json.Unmarshal(r, &out[i]); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

const fightWithVictoryTypes = `{"id":"f","victoryTypesArray":[{"identifier":"VFA","name":"VICTORY BY FALL"},{"identifier":"VPO1","name":"VICTORY BY POINTS"}]}`

func newMat() *MatState {
	return &MatState{Fight: json.RawMessage(fightWithVictoryTypes), Control: map[string]json.RawMessage{}}
}

func TestAutoVictoryFromRealTimingData(t *testing.T) {
	msgs := loadTimingFixture(t)
	s := &Syncer{}
	m := newMat()
	now := time.Now()

	var victoryAt = -1
	for i, lm := range msgs {
		s.autoFromLive(m, lm, now)
		if _, ok := m.Control["victory"]; ok && victoryAt < 0 {
			victoryAt = i
		}
	}
	vic, ok := controlOf(m, "victory")
	if !ok {
		t.Fatal("vitória não foi acionada com os dados reais")
	}
	// No dado real: winner chega primeiro com victory vazio; só aciona quando o tipo vem.
	if d := msgs[victoryAt].Message.Detail; d == nil || d.Victory == "" {
		t.Fatalf("acionou antes do tipo de vitória: %+v", d)
	}
	if vic.Side != "home" || vic.VictoryType != "VICTORY BY FALL" || !vic.Auto {
		t.Fatalf("vitória errada: %+v", vic)
	}
}

func TestAutoIntroStartingThenRunning(t *testing.T) {
	s := &Syncer{} // sem Settings = apresentação automática ligada
	m := newMat()
	now := time.Now()
	var lm liveMessage
	lm.Message.ID = "fight-1"

	lm.State = "starting"
	if !s.autoFromLive(m, lm, now) {
		t.Fatal("starting deveria mostrar a apresentação")
	}
	if _, ok := m.Control["intro"]; !ok {
		t.Fatal("apresentação não entrou")
	}
	// mensagens repetidas de "starting" não reaplicam
	if s.autoFromLive(m, lm, now) {
		t.Fatal("não deveria mudar de novo")
	}
	lm.State = "running"
	s.autoFromLive(m, lm, now)
	if _, ok := m.Control["intro"]; ok {
		t.Fatal("apresentação deveria sair quando o cronômetro inicia")
	}
	// a mesma luta voltando a "starting" (ex.: pausa) não reapresenta
	lm.State = "starting"
	s.autoFromLive(m, lm, now)
	if _, ok := m.Control["intro"]; ok {
		t.Fatal("não deveria reapresentar a mesma luta")
	}
}

func TestManualIntroIsNotRemovedByRunning(t *testing.T) {
	s := &Syncer{}
	m := newMat()
	b, _ := json.Marshal(controlState{ControlRequest: ControlRequest{Graphic: "intro", Visible: true}})
	m.Control["intro"] = b // acionada pelo operador (Auto=false)
	var lm liveMessage
	lm.State, lm.Message.ID = "running", "f"
	s.autoFromLive(m, lm, time.Now())
	if _, ok := m.Control["intro"]; !ok {
		t.Fatal("apresentação manual não pode sair sozinha")
	}
}

func TestVictoryName(t *testing.T) {
	if got := victoryName(json.RawMessage(fightWithVictoryTypes), "VFA"); got != "VICTORY BY FALL" {
		t.Fatal(got)
	}
	if got := victoryName(json.RawMessage(`null`), "XYZ"); got != "XYZ" {
		t.Fatal(got)
	}
}
