package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func fakeKey(k string, err error) func(context.Context, string) (string, error) {
	return func(context.Context, string) (string, error) { return k, err }
}

func TestSetupCreatesEnv(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	in := strings.NewReader("http://arena:8080/\napp1\nsecret1\nkey1\n\n192.168.0.11\n")
	var out bytes.Buffer
	err := runSetup(in, &out, setupOpts{EnvPath: env, ArenaDir: dir, MercureKey: fakeKey("mk", nil), Rand: rand.Reader})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(env)
	s := string(b)
	for _, want := range []string{
		"ARENA_URL=http://arena:8080\n", "ARENA_CLIENT_ID=app1\n", "ARENA_CLIENT_SECRET=secret1\n",
		"ARENA_API_KEY=key1\n", "MERCURE_HUB_URL=http://arena:8080/.well-known/mercure\n",
		"MERCURE_JWT_KEY=mk\n", "TIMING_ADDRS=192.168.0.11\n",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("falta %q em:\n%s", want, s)
		}
	}
	if strings.Contains(out.String(), "secret1") || strings.Contains(out.String(), "key1") {
		t.Error("segredo apareceu na saída")
	}
	if runtime.GOOS != "windows" {
		if st, _ := os.Stat(env); st.Mode().Perm() != 0o600 {
			t.Errorf("permissão %v", st.Mode().Perm())
		}
	}
}

func TestSetupWebhookTokenIsRandom(t *testing.T) {
	read := func() string {
		dir := t.TempDir()
		env := filepath.Join(dir, ".env")
		in := strings.NewReader("\napp\nsec\nkey\n\n\n")
		if err := runSetup(in, &bytes.Buffer{}, setupOpts{EnvPath: env, ArenaDir: dir, MercureKey: fakeKey("mk", nil), Rand: rand.Reader}); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(env)
		for _, l := range strings.Split(string(b), "\n") {
			if v, ok := strings.CutPrefix(l, "WEBHOOK_TOKEN="); ok {
				return v
			}
		}
		return ""
	}
	a, b := read(), read()
	if len(a) != 48 || a == b {
		t.Fatalf("token fraco: %q %q", a, b)
	}
}

func TestSetupKeepsExistingEnvUnlessConfirmed(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	os.WriteFile(env, []byte("ORIGINAL=1\n"), 0o600)
	var out bytes.Buffer
	if err := runSetup(strings.NewReader("n\n"), &out, setupOpts{EnvPath: env, Rand: rand.Reader}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(env); string(b) != "ORIGINAL=1\n" {
		t.Fatalf("arquivo alterado: %q", b)
	}
	// EOF sem resposta também não sobrescreve
	if err := runSetup(strings.NewReader(""), &out, setupOpts{EnvPath: env, Rand: rand.Reader}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(env); string(b) != "ORIGINAL=1\n" {
		t.Fatal("EOF sobrescreveu")
	}
}

func TestSetupOverwriteConfirmedTightensMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("modo Unix")
	}
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	os.WriteFile(env, []byte("OLD=1\n"), 0o644)
	in := strings.NewReader("s\n\napp\nsec\nkey\n\n\n")
	if err := runSetup(in, &bytes.Buffer{}, setupOpts{EnvPath: env, ArenaDir: dir, MercureKey: fakeKey("mk", nil), Rand: rand.Reader}); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(env); st.Mode().Perm() != 0o600 {
		t.Fatalf("permissão %v", st.Mode().Perm())
	}
}

func TestSetupAsksMercureKeyWhenDockerFails(t *testing.T) {
	dir := t.TempDir()
	env := filepath.Join(dir, ".env")
	in := strings.NewReader("\napp\nsec\nkey\n\n\nchave-manual\n")
	var out bytes.Buffer
	err := runSetup(in, &out, setupOpts{EnvPath: env, ArenaDir: dir, MercureKey: fakeKey("", errors.New("sem docker")), Rand: rand.Reader})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(env); !strings.Contains(string(b), "MERCURE_JWT_KEY=chave-manual\n") {
		t.Fatalf("chave manual ausente:\n%s", b)
	}
}

func TestSetupRejectsMissingRequiredOnEOF(t *testing.T) {
	err := runSetup(strings.NewReader("\n"), &bytes.Buffer{}, setupOpts{EnvPath: filepath.Join(t.TempDir(), ".env"), Rand: rand.Reader})
	if err == nil {
		t.Fatal("esperava erro por EOF em campo obrigatório")
	}
}
