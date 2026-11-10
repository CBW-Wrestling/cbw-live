package live

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"cbwlive/internal/mercure"
)

func TestSettingsPersistAndValidate(t *testing.T) {
	hubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer hubSrv.Close()
	hub := mercure.NewHub(hubSrv.URL, "k")
	dir := t.TempDir()

	st, err := NewSettingsStore(dir, hub)
	if err != nil {
		t.Fatal(err)
	}
	if got := st.Get(); got != DefaultSettings() || got.ShowStartIn {
		t.Fatalf("padrão inesperado: %+v", got)
	}

	if err := st.Set(context.Background(), Settings{CollapseMode: "bogus"}); err == nil {
		t.Fatal("deveria rejeitar collapseMode inválido")
	}
	want := Settings{CollapseMode: "never", AfterScoreSeconds: 3, ShowStartIn: true, ShowEventInfo: false, Background: "green",
		Scoreboard: "judo", Position: "bottom-end", IntroPosition: "top-start", IntroStyle: "split", Anim: "wipe"}
	if err := st.Set(context.Background(), want); err != nil {
		t.Fatal(err)
	}

	// recarrega do disco
	st2, err := NewSettingsStore(dir, hub)
	if err != nil {
		t.Fatal(err)
	}
	if st2.Get() != want {
		t.Fatalf("não persistiu: %+v", st2.Get())
	}

	// arquivo antigo sem um campo novo: o campo fica com o valor padrão
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"collapseMode":"always"}`), 0o644)
	st3, _ := NewSettingsStore(dir, hub)
	if g := st3.Get(); g.CollapseMode != "always" || g.AfterScoreSeconds != 6 || !g.ShowEventInfo || g.Background != "transparent" || g.Scoreboard != "classic" || g.Position != "top-start" || g.IntroPosition != "auto" {
		t.Fatalf("merge com padrão falhou: %+v", g)
	}

	// valor antigo de posição é convertido
	os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"position":"center"}`), 0o644)
	st4, _ := NewSettingsStore(dir, hub)
	if g := st4.Get(); g.Position != "top-center" {
		t.Fatalf("posição antiga não migrou: %+v", g)
	}
}
