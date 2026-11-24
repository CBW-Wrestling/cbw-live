package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"
)

// setupMain implementa `cbwlive setup`: gera o .env de forma interativa,
// igual em macOS, Windows e Linux (substitui scripts/setup-env.sh).
func setupMain(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	envPath := fs.String("env", ".env", "arquivo .env a criar")
	home, _ := os.UserHomeDir()
	arenaDir := fs.String("arena-dir", filepath.Join(home, "Downloads", "Arena"), "pasta do docker-compose do Arena (para ler a chave do Mercure)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return runSetup(os.Stdin, os.Stdout, setupOpts{
		EnvPath:    *envPath,
		ArenaDir:   *arenaDir,
		MercureKey: mercureKeyFromDocker,
		Rand:       rand.Reader,
	})
}

type setupOpts struct {
	EnvPath    string
	ArenaDir   string
	MercureKey func(ctx context.Context, arenaDir string) (string, error)
	Rand       io.Reader
}

type prompter struct {
	in  *bufio.Reader
	out io.Writer
	fd  int // descritor do terminal; -1 se a entrada não é um terminal
}

func newPrompter(in io.Reader, out io.Writer) *prompter {
	p := &prompter{in: bufio.NewReader(in), out: out, fd: -1}
	if f, ok := in.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		p.fd = int(f.Fd())
	}
	return p
}

func (p *prompter) line() (string, error) {
	s, err := p.in.ReadString('\n')
	if err != nil && (err != io.EOF || s == "") {
		return "", err
	}
	return strings.TrimSpace(s), nil
}

func (p *prompter) ask(label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(p.out, "%s [%s]: ", label, def)
	} else {
		fmt.Fprintf(p.out, "%s: ", label)
	}
	s, err := p.line()
	if err != nil {
		return "", err
	}
	if s == "" {
		s = def
	}
	return s, checkValue(label, s)
}

// askSecret não ecoa o que é digitado quando a entrada é um terminal.
func (p *prompter) askSecret(label string) (string, error) {
	fmt.Fprintf(p.out, "%s: ", label)
	var s string
	if p.fd >= 0 {
		b, err := term.ReadPassword(p.fd)
		fmt.Fprintln(p.out)
		if err != nil {
			return "", err
		}
		s = strings.TrimSpace(string(b))
	} else {
		var err error
		if s, err = p.line(); err != nil {
			return "", err
		}
	}
	return s, checkValue(label, s)
}

func checkValue(label, s string) error {
	if strings.ContainsAny(s, "\r\n") {
		return fmt.Errorf("%s: valor inválido (quebra de linha)", label)
	}
	return nil
}

func (p *prompter) required(label string, secret bool) (string, error) {
	for {
		var s string
		var err error
		if secret {
			s, err = p.askSecret(label)
		} else {
			s, err = p.ask(label, "")
		}
		if err != nil {
			return "", err
		}
		if s != "" {
			return s, nil
		}
		fmt.Fprintln(p.out, "  obrigatório.")
	}
}

func isYes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes", "s", "sim":
		return true
	}
	return false
}

func runSetup(in io.Reader, out io.Writer, o setupOpts) error {
	p := newPrompter(in, out)

	if _, err := os.Stat(o.EnvPath); err == nil {
		fmt.Fprintf(out, "%s já existe. Sobrescrever? [s/N] ", o.EnvPath)
		ans, err := p.line()
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if !isYes(ans) {
			fmt.Fprintln(out, "Nada alterado.")
			return nil
		}
	}

	url, err := p.ask("Arena URL", "http://localhost:8080")
	if err != nil {
		return err
	}
	url = strings.TrimRight(url, "/")
	cid, err := p.required("App ID (Settings → Apps)", false)
	if err != nil {
		return err
	}
	csec, err := p.required("App Secret", true)
	if err != nil {
		return err
	}
	akey, err := p.required("API Key do usuário de placar (Settings → Users)", true)
	if err != nil {
		return err
	}
	ev, err := p.ask("ID do evento (vazio = todos)", "")
	if err != nil {
		return err
	}
	timing, err := p.ask("IPs dos PCs do UWW Timing, separados por vírgula (vazio = nenhum)", "")
	if err != nil {
		return err
	}

	mkey := ""
	if o.MercureKey != nil && o.ArenaDir != "" {
		if st, err := os.Stat(o.ArenaDir); err == nil && st.IsDir() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			mkey, err = o.MercureKey(ctx, o.ArenaDir)
			cancel()
			if err != nil {
				mkey = ""
			}
		}
	}
	if mkey == "" {
		fmt.Fprintln(out, "Não consegui ler a chave do Mercure pelo Docker. Informe MERCURE_PUBLISHER_JWT_KEY do container caddy do Arena.")
		if mkey, err = p.required("MERCURE_PUBLISHER_JWT_KEY", true); err != nil {
			return err
		}
	}

	tok := make([]byte, 24)
	if _, err := io.ReadFull(o.Rand, tok); err != nil {
		return err
	}

	content := fmt.Sprintf(`ARENA_URL=%s
ARENA_CLIENT_ID=%s
ARENA_CLIENT_SECRET=%s
ARENA_API_KEY=%s
ARENA_EVENT_ID=%s
MERCURE_HUB_URL=%s/.well-known/mercure
MERCURE_JWT_KEY=%s
LISTEN_ADDR=:8095
DATA_DIR=data
WEBHOOK_TOKEN=%s
WEBHOOK_TOKEN_ENFORCE=false
RESYNC_SECONDS=15
TIMING_ADDRS=%s
`, url, cid, csec, akey, ev, url, mkey, hex.EncodeToString(tok), timing)

	f, err := os.OpenFile(o.EnvPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	_ = os.Chmod(o.EnvPath, 0o600) // arquivo pré-existente mantém o modo antigo sem isto

	if runtime.GOOS == "windows" {
		fmt.Fprintf(out, "%s criado. Ele contém segredos: não compartilhe nem versione.\n", o.EnvPath)
	} else {
		fmt.Fprintf(out, "%s criado (permissão 600).\n", o.EnvPath)
	}
	return nil
}

// mercureKeyFromDocker lê MERCURE_PUBLISHER_JWT_KEY do container caddy do Arena.
func mercureKeyFromDocker(ctx context.Context, arenaDir string) (string, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "docker", "compose", "exec", "-T", "caddy", "sh", "-c", `printf %s "$MERCURE_PUBLISHER_JWT_KEY"`)
	cmd.Dir = arenaDir
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	k := strings.TrimSpace(string(b))
	if k == "" {
		return "", errors.New("chave vazia")
	}
	return k, nil
}
