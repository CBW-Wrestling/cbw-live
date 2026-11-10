// arenaget: consulta a API do Arena com as credenciais do .env (ferramenta de desenvolvimento).
//
//	go run ./cmd/arenaget scoreboard/sport-event
//	go run ./cmd/arenaget team/<sportEventId>
package main

import (
	"context"
	"fmt"
	"os"

	"cbwlive/internal/arena"
	"cbwlive/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: arenaget <caminho depois de /api/json/>")
		os.Exit(2)
	}
	cfg, err := config.Load(".env")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	c := arena.New(cfg.ArenaURL, cfg.ArenaClientID, cfg.ArenaClientSecret, cfg.ArenaAPIKey)
	body, err := c.GetRaw(context.Background(), os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(body)
}
