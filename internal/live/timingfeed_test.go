package live

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"cbwlive/internal/mercure"
)

// Reproduz a sessão real gravada do UWW Timing (10/11/2026) no timingFeed e
// confere a sequência de gráficos automáticos que o overlay recebe.
func TestTimingFeedReplaysRealSession(t *testing.T) {
	hub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer hub.Close()

	const eventID = "1f1dc89e-e944-67e0-b4bc-4d7c3dcfa26c"
	s := NewSyncer(nil, mercure.NewHub(hub.URL, "k"), NewStore(), "", 0, nil)
	s.Store.Update(Key(eventID, "A"), func(m *MatState) bool {
		m.EventID, m.MatName = eventID, "A"
		m.Fight = json.RawMessage(`{"id":"x","victoryTypesArray":[{"identifier":"VFA","name":"VICTORY BY FALL"}]}`)
		return true
	})

	b, err := os.ReadFile("../timing/testdata/session.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw []string
	json.Unmarshal(b, &raw)

	f := &timingFeed{s: s, addr: "test:9696"}
	var seq []string
	last := ""
	for _, r := range raw {
		f.handle(context.Background(), r)
		st, _ := s.Store.Get(Key(eventID, "A"))
		var parts []string
		for _, g := range []string{"intro", "challenge", "injury", "passivity", "victory"} {
			if cs, ok := controlOf(&st, g); ok {
				p := g
				if cs.Side != "" {
					p += ":" + cs.Side
				}
				if cs.Status != "" {
					p += ":" + cs.Status
				}
				if cs.VictoryType != "" {
					p += ":" + cs.VictoryType
				}
				parts = append(parts, p)
			}
		}
		now := strings.Join(parts, ",")
		if now != last {
			seq = append(seq, now)
			last = now
		}
	}
	got := strings.Join(seq, " | ")
	t.Log(got)

	if !s.timingActive(Key(eventID, "A")) {
		t.Fatal("o Timing deveria estar ligado ao tapete A (via mch)")
	}
	for _, want := range []string{
		"intro",                        // pre (luta carregada)
		"challenge:home:requested",     // ch1;
		"challenge:home:challenge",     // ch1;1
		"challenge:home:won",           // ch1;1;1
		"challenge:away:requested",     // ch2;
		"challenge:away:canceled",      // ch2;0
		"challenge:away:lost",          // ch2;1;0
		"injury:away",                  // ij2 start/show
		"passivity:home",               // ac1 show
		"victory:home:VICTORY BY FALL", // wmh;GUIDEA I.;5-0VFA
	} {
		if !strings.Contains(got, want) {
			t.Errorf("faltou %q na sequência", want)
		}
	}
}

func TestTimingLiveJSONHasOverlayFormat(t *testing.T) {
	f := &timingFeed{eventID: "e", matName: "A", fightID: "f", clock: "5:58", started: true,
		home: timingAthlete{Name: "GUIDEA I.", Country: "ROU", Score: 4}, away: timingAthlete{Name: "TOMAR S.", Country: "IND"}, advantage: "home"}
	var lm liveMessage
	if err := json.Unmarshal(f.liveJSON(), &lm); err != nil {
		t.Fatal(err)
	}
	if lm.State != "running" || lm.Message.ID != "f" || lm.Message.Home.Code != "" {
		t.Fatalf("formato inesperado: %+v", lm)
	}
}
