// cbwlive: serviço de integração UWW Arena → gráficos/placares da CBW.
//
//	Arena (webhook POST) ─┐
//	Arena (REST/OAuth)  ──┼─▶ cbwlive ──publish──▶ Mercure ──SSE──▶ overlays / telões
//	Mercure (placar UWW) ─┘
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"cbwlive/internal/arena"
	"cbwlive/internal/config"
	"cbwlive/internal/hubproxy"
	"cbwlive/internal/live"
	"cbwlive/internal/mercure"
	"cbwlive/internal/webhook"
	"cbwlive/web"
)

func main() {
	envFile := flag.String("env", ".env", "arquivo de configuração")
	debug := flag.Bool("debug", false, "log detalhado")
	flag.Parse()

	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	if err := run(*envFile); err != nil {
		slog.Error("encerrando", "err", err)
		os.Exit(1)
	}
}

func run(envFile string) error {
	cfg, err := config.Load(envFile)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	webhookLog, err := live.NewRecorder(cfg.DataDir, "webhooks.jsonl")
	if err != nil {
		return err
	}
	defer webhookLog.Close()
	liveLog, err := live.NewRecorder(cfg.DataDir, "uww-live.jsonl")
	if err != nil {
		return err
	}
	defer liveLog.Close()

	arenaClient := arena.New(cfg.ArenaURL, cfg.ArenaClientID, cfg.ArenaClientSecret, cfg.ArenaAPIKey)
	hub := mercure.NewHub(cfg.MercureHubURL, cfg.MercureJWTKey)
	store := live.NewStore()
	syncer := live.NewSyncer(arenaClient, hub, store, cfg.ArenaEventID, cfg.Resync, liveLog)
	defer syncer.Close()

	settings, err := live.NewSettingsStore(cfg.DataDir, hub)
	if err != nil {
		return err
	}

	syncer.Settings = settings

	proxy, err := hubproxy.New(cfg.MercureHubURL, hub.SubscriberToken())
	if err != nil {
		return err
	}
	static, _ := fs.Sub(web.Files, ".")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Cbwlive", "1")
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, store.All()) })
	mux.HandleFunc("GET /api/state/{event}/{mat}", func(w http.ResponseWriter, r *http.Request) {
		st, ok := store.Get(live.Key(r.PathValue("event"), r.PathValue("mat")))
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, st)
	})
	mux.HandleFunc("POST /api/control/{event}/{mat}", func(w http.ResponseWriter, r *http.Request) {
		var req live.ControlRequest
		if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		st, err := syncer.ApplyControl(r.Context(), r.PathValue("event"), r.PathValue("mat"), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.Info("controle", "mat", r.PathValue("mat"), "graphic", req.Graphic, "visible", req.Visible, "side", req.Side, "status", req.Status)
		writeJSON(w, st)
	})
	mux.HandleFunc("GET /api/settings", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, settings.Get()) })
	mux.HandleFunc("PUT /api/settings", func(w http.ResponseWriter, r *http.Request) {
		var in live.Settings
		if err := json.NewDecoder(io.LimitReader(r.Body, 16<<10)).Decode(&in); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		if err := settings.Set(r.Context(), in); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		slog.Info("configurações salvas", "collapseMode", in.CollapseMode, "showStartIn", in.ShowStartIn)
		writeJSON(w, settings.Get())
	})
	mux.Handle("GET /flags/{code}", newFlagProxy(cfg.ArenaURL, filepath.Join(cfg.AssetsDir, "flags")))
	// logo e demais arquivos visuais da CBW (assets/logo.svg, …)
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", noCache(http.FileServer(http.Dir(cfg.AssetsDir)))))
	mux.Handle("GET /hub", proxy)
	mux.Handle("POST /webhook/arena", &webhook.Handler{
		Token: cfg.WebhookToken, Enforce: cfg.WebhookTokenEnforce, Log: webhookLog,
		OnEvent: func(entity, action, id string) {
			syncer.Trigger()
			go syncer.HandleEntity(context.WithoutCancel(ctx), entity, action, id)
		},
	})
	mux.Handle("GET /", http.FileServerFS(static))

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// sem WriteTimeout: /hub mantém conexões SSE abertas
	}

	allLog, err := live.NewRecorder(cfg.DataDir, "mercure-all.jsonl")
	if err != nil {
		return err
	}
	defer allLog.Close()

	go syncer.Run(ctx)
	go syncer.SniffAll(ctx, allLog)

	// conexão direta com cada UWW Timing (challenge em etapas, sangue, passividade, pausa…)
	if len(cfg.TimingAddrs) > 0 {
		timingLog, err := live.NewRecorder(cfg.DataDir, "timing-ws.jsonl")
		if err != nil {
			return err
		}
		defer timingLog.Close()
		for _, addr := range cfg.TimingAddrs {
			go syncer.RunTimingFeed(ctx, addr, timingLog)
		}
		slog.Info("UWW Timing direto", "pcs", cfg.TimingAddrs)
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(shutdownCtx)
	}()

	ln, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return fmt.Errorf("não consegui abrir %s: %w", cfg.ListenAddr, err)
	}
	go checkPortOwnership(cfg.ListenAddr)

	slog.Info("cbwlive no ar", "addr", cfg.ListenAddr, "arena", cfg.ArenaURL, "evento", orAll(cfg.ArenaEventID))
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// checkPortOwnership confirma, por IPv4, que é o cbwlive quem responde na porta.
// No macOS outro programa pode ocupar a mesma porta só em IPv4 (foi o caso do
// REAPER na 8090): o Mac chegava ao cbwlive, mas o Docker caía no outro app.
func checkPortOwnership(addr string) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return
	}
	time.Sleep(200 * time.Millisecond)
	resp, err := (&http.Client{Timeout: 2 * time.Second}).Get("http://127.0.0.1:" + port + "/healthz")
	if err != nil {
		slog.Warn("checagem de porta: sem resposta via IPv4", "porta", port, "err", err)
		return
	}
	resp.Body.Close()
	if resp.Header.Get("X-Cbwlive") != "1" {
		slog.Error("OUTRO PROGRAMA responde nesta porta via IPv4 — webhooks do Docker não chegarão ao cbwlive; troque LISTEN_ADDR",
			"porta", port, "server", resp.Header.Get("Server"), "status", resp.StatusCode)
	}
}

// newFlagProxy serve as bandeiras SVG que o próprio Arena já hospeda
// (/build/images/flags/custom/{código}.svg), com cache em memória — funciona offline no ginásio.
//
// Bandeiras em localDir (ex.: estados brasileiros, mesmos SVGs do UWW Timing) têm
// prioridade e não ficam em cache, para poderem ser trocadas com o serviço rodando.
func newFlagProxy(arenaURL, localDir string) http.Handler {
	var mu sync.Mutex
	cache := map[string][]byte{}
	valid := regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{1,23}\.svg$`)
	client := &http.Client{Timeout: 3 * time.Second}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := strings.ToLower(r.PathValue("code"))
		if !strings.HasSuffix(code, ".svg") {
			code += ".svg"
		}
		if !valid.MatchString(code) {
			http.NotFound(w, r)
			return
		}
		if b, err := os.ReadFile(filepath.Join(localDir, code)); err == nil {
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(b)
			return
		}
		mu.Lock()
		body, ok := cache[code]
		mu.Unlock()
		if !ok {
			resp, err := client.Get(arenaURL + "/build/images/flags/custom/" + code)
			if err != nil {
				http.Error(w, "arena indisponível", http.StatusBadGateway)
				return
			}
			body, _ = io.ReadAll(io.LimitReader(resp.Body, 512<<10))
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK || !bytes.Contains(body, []byte("<svg")) {
				http.NotFound(w, r)
				return
			}
			mu.Lock()
			cache[code] = body
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(body)
	})
}

func noCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		h.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func orAll(s string) string {
	if s == "" {
		return "todos"
	}
	return s
}
