package live

import (
	"encoding/json"
	"testing"
)

func TestPickFight(t *testing.T) {
	fights := []json.RawMessage{
		json.RawMessage(`{"id":"f1","fightNumber":1}`),
		json.RawMessage(`{"id":"f2","fightNumber":2}`),
	}
	cases := map[string]string{"": "f1", "f2": "f2", "desconhecida": "f1"}
	for liveID, want := range cases {
		var got struct{ ID string }
		json.Unmarshal(pickFight(fights, liveID), &got)
		if got.ID != want {
			t.Errorf("liveID=%q: escolheu %q, esperado %q", liveID, got.ID, want)
		}
	}
	if string(pickFight(nil, "f1")) != "null" {
		t.Error("sem lutas deveria ser null")
	}
}

func TestLiveMessageParsing(t *testing.T) {
	// payload real capturado do UWW Timing (tópico http://arena.uww.org/live/{evento}/{tapete})
	raw := `{"sportEventId":"e","state":"running","matName":"A","message":{"id":"f1","idle":false,"time":"5:55",
		"home":{"name":"RAHIMI H.","score":4},"away":{"name":"LEBEDEV V.","score":0},"advantage":""}}`
	var lm liveMessage
	if err := json.Unmarshal([]byte(raw), &lm); err != nil {
		t.Fatal(err)
	}
	if lm.State != "running" || lm.Message.ID != "f1" || lm.Message.Idle {
		t.Fatalf("parse errado: %+v", lm)
	}
}
