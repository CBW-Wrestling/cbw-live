package live

import (
	"testing"
	"time"
)

func TestActiveAreaFollowsEvent(t *testing.T) {
	s := NewSyncer(nil, nil, NewStore(), "", 0, nil)
	set := func(ev string, fn func(m *MatState)) {
		s.Store.Update(Key(ev, "A"), func(m *MatState) bool { m.EventID, m.MatName = ev, "A"; fn(m); return true })
	}
	want := func(ev string) {
		t.Helper()
		st, ok := s.ActiveArea("A")
		if !ok || st.EventID != ev {
			t.Fatalf("ativo = %q, quero %q", st.EventID, ev)
		}
	}
	early, late := time.Now().Add(-time.Hour), time.Now()

	set("e1", func(m *MatState) { m.OpenFights = 3 })
	set("e2", func(m *MatState) {})
	want("e1")
	set("e1", func(m *MatState) { m.LiveAt = &early })
	set("e2", func(m *MatState) { m.OpenFights = 2 })
	want("e1")
	set("e2", func(m *MatState) { m.LiveAt = &late })
	want("e2")
	set("e1", func(m *MatState) { m.TimingAddr = "x:9696" })
	want("e1")
	s.listed = map[string]bool{Key("e2", "A"): true} // e1 saiu do Arena
	want("e2")

	if _, ok := s.ActiveArea("B"); ok {
		t.Fatal("tapete B não existe")
	}
}
