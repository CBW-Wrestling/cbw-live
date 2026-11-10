// timingtap: escuta o WebSocket do UWW Timing (UWW TV Scoreboard, porta 9696) e
// grava cada mensagem com horário — ferramenta de descoberta do protocolo.
//
//	go run ./cmd/timingtap -addr 192.168.0.11:9696
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"cbwlive/internal/timing"
)

func main() {
	addr := flag.String("addr", "", "host:porta do UWW Timing (ex.: 192.168.0.11:9696)")
	out := flag.String("out", "data/timing-ws.jsonl", "arquivo de saída (JSON Lines)")
	flag.Parse()
	if *addr == "" {
		fmt.Fprintln(os.Stderr, "uso: timingtap -addr <ip-do-timing>:9696")
		os.Exit(2)
	}
	os.MkdirAll(filepath.Dir(*out), 0o755)
	f, err := os.OpenFile(*out, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	timing.Listen(ctx, *addr, func(msg string) {
		now := time.Now()
		line, _ := json.Marshal(map[string]string{"at": now.Format(time.RFC3339Nano), "msg": msg})
		f.Write(append(line, '\n'))
		fmt.Printf("%s  %s\n", now.Format("15:04:05.000"), msg)
	}, nil)
}
