package live

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Recorder grava cada mensagem recebida em JSON Lines, para descobrirmos
// os formatos reais dos payloads do Arena (webhooks e Mercure).
type Recorder struct {
	mu sync.Mutex
	f  *os.File
}

func NewRecorder(dir, name string) (*Recorder, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	return &Recorder{f: f}, nil
}

func (r *Recorder) Write(fields map[string]any) {
	fields["at"] = time.Now().Format(time.RFC3339Nano)
	line, err := json.Marshal(fields)
	if err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.f.Write(append(line, '\n'))
}

func (r *Recorder) Close() error { return r.f.Close() }

// RawOrString devolve o corpo como JSON se for válido; senão como string.
func RawOrString(b []byte) any {
	if json.Valid(b) {
		return json.RawMessage(b)
	}
	return string(b)
}
