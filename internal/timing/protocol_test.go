package timing

import (
	"encoding/json"
	"os"
	"testing"
)

func TestParseState(t *testing.T) {
	m := Parse("1;6;Seniors FS 74;Seniors;5:11;AUS;ILYASOV T.;BUL;IVANOV G.;0;4;0;0;0;0;30;4:00;30;3:52;0;-1;;Qualif.;#ba0000;#0000aa;180;2;30;")
	s := m.State
	if m.Head != "state" || s == nil {
		t.Fatalf("não reconheceu o estado: %+v", m)
	}
	if s.MatchNo != "6" || s.Clock != "5:11" || s.HomeName != "ILYASOV T." || s.AwayCountry != "BUL" ||
		s.AwayScore != 4 || s.Winner != -1 || s.Round != "Qualif." || s.RoundDuration != 180 || s.Rounds != 2 {
		t.Fatalf("campos errados: %+v", s)
	}
}

func TestParseEvents(t *testing.T) {
	cases := []struct {
		raw, head, status, side string
	}{
		{"evt;ch1;;applyAt=1794302074320", "ch1", "requested", "home"},
		{"evt;ch1;1;applyAt=1", "ch1", "challenge", "home"},
		{"evt;ch1;1;1;applyAt=1", "ch1", "won", "home"},
		{"evt;ch2;1;0;applyAt=1", "ch2", "lost", "away"},
		{"evt;ch2;0;applyAt=1", "ch2", "canceled", "away"},
	}
	for _, c := range cases {
		m := Parse(c.raw)
		if m.Head != c.head || m.ChallengeStatus() != c.status || m.Side() != c.side {
			t.Errorf("%s → head=%s status=%s side=%s", c.raw, m.Head, m.ChallengeStatus(), m.Side())
		}
	}
	if m := Parse("evt;sc1;4;sc2;0;applyAt=1"); true {
		x, y, ok := m.Pair()
		if !ok || x != 4 || y != 0 {
			t.Errorf("sc1: %d %d %v", x, y, ok)
		}
	}
	if m := Parse("evt;clk;5:58;stop;applyAt=1"); m.Arg(0) != "5:58" || m.Arg(1) != "stop" {
		t.Errorf("clk: %+v", m)
	}
	if n, c := SplitNameCountry("GOZIUMOV K.(AZE)"); n != "GOZIUMOV K." || c != "AZE" {
		t.Errorf("nome/país: %q %q", n, c)
	}
	if v := VictoryCode("5-0VFA"); v != "VFA" {
		t.Errorf("vitória: %q", v)
	}
	if v := VictoryCode("12-2VSU1"); v != "VSU1" {
		t.Errorf("vitória com dígito: %q", v)
	}
}

// Sessão real gravada do UWW Timing (10/11/2026): todas as mensagens são reconhecidas.
func TestRealSessionIsRecognized(t *testing.T) {
	b, err := os.ReadFile("testdata/session.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw []string
	json.Unmarshal(b, &raw)
	known := map[string]bool{"state": true, "clk": true, "pre": true, "rdy": true, "mch": true, "wt1": true,
		"sc1": true, "ct1": true, "avt": true, "ch1": true, "ch2": true, "ij1": true, "ij2": true,
		"ac1": true, "ac2": true, "wmh": true, "bye": true, "brk": true}
	seen := map[string]int{}
	for _, r := range raw {
		m := Parse(r)
		if !known[m.Head] {
			t.Errorf("mensagem não reconhecida: %q", r)
		}
		seen[m.Head]++
	}
	for _, h := range []string{"state", "mch", "pre", "rdy", "clk", "ch1", "ch2", "ij1", "ij2", "ac1", "wmh"} {
		if seen[h] == 0 {
			t.Errorf("a sessão gravada deveria ter %q", h)
		}
	}
}
