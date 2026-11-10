// Package timing conecta ao "UWW TV Scoreboard" que o próprio UWW Timing serve
// (HTTP em :9696, WebSocket em /ws) e entrega cada mensagem de texto recebida.
//
// Protocolo (texto separado por ";", descoberto em 10/11/2026 lendo a página
// que o Timing serve em http://<timing>:9696/):
//
//	<estado completo>            "1;6;Seniors FS 74;Seniors;5:11;AUS;ILYASOV T.;BUL;IVANOV G.;0;4;…"
//	evt;clk;<T>;start|stop|stopend   cronômetro
//	evt;pre / evt;rdy                pré-luta (match ready) / pronta
//	evt;ij1|ij2;<T>;<cmd>            atendimento (sangue) vermelho/azul
//	evt;ac1|ac2;<T>;<cmd>            passividade (activity) vermelho/azul
//	evt;ch1|ch2;<a>;<b>              challenge: ""=pedido, "1"=aceito, "1;1"=ganhou, "1;0"=perdeu, "0"=cancelado
//	evt;wmh;<vencedor>;<tipo>        vitória
//	evt;brk;<T>;<cmd>                intervalo
//
// Cliente WebSocket mínimo (RFC 6455) só com a biblioteca padrão: o servidor do
// Timing manda texto; respondemos ping com pong e reconectamos sozinhos.
package timing

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Com o Timing ocioso (sem luta correndo) não chega nada; mandamos ping a cada
// pingEvery e consideramos a conexão morta se nem o pong voltar em readTimeout.
const (
	pingEvery   = 15 * time.Second
	readTimeout = 45 * time.Second
)

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Listen conecta em ws://addr/ws e chama fn para cada mensagem, reconectando
// com backoff até ctx terminar. addr é "host:porta" (ex.: 192.168.0.11:9696).
// onConn (opcional) é avisado ao conectar (true) e ao perder a conexão (false).
func Listen(ctx context.Context, addr string, fn func(msg string), onConn func(connected bool)) {
	backoff := time.Second
	for ctx.Err() == nil {
		err := session(ctx, addr, fn, func() {
			backoff = time.Second
			if onConn != nil {
				onConn(true)
			}
		})
		if onConn != nil {
			onConn(false)
		}
		if ctx.Err() != nil {
			return
		}
		slog.Warn("timing: conexão encerrada, reconectando", "addr", addr, "err", err, "em", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 10*time.Second)
	}
}

func session(ctx context.Context, addr string, fn func(string), connected func()) error {
	d := net.Dialer{Timeout: 5 * time.Second, KeepAlive: 15 * time.Second}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	go func() { <-ctx.Done(); conn.Close() }()

	keyRaw := make([]byte, 16)
	rand.Read(keyRaw)
	key := base64.StdEncoding.EncodeToString(keyRaw)
	req := "GET /ws HTTP/1.1\r\nHost: " + addr + "\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\nSec-WebSocket-Version: 13\r\n\r\n"
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.WriteString(conn, req); err != nil {
		return err
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSwitchingProtocols {
		return fmt.Errorf("handshake: HTTP %d", resp.StatusCode)
	}
	sum := sha1.Sum([]byte(key + wsGUID))
	if resp.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(sum[:]) {
		return errors.New("handshake: Sec-WebSocket-Accept inválido")
	}
	conn.SetDeadline(time.Time{})
	slog.Info("timing: conectado", "addr", addr)
	connected()

	var wmu sync.Mutex // ping (goroutine) e pong (leitura) escrevem na mesma conexão
	write := func(op byte, p []byte) error {
		wmu.Lock()
		defer wmu.Unlock()
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		return writeFrame(conn, op, p)
	}
	stopPing := make(chan struct{})
	defer close(stopPing)
	go func() {
		t := time.NewTicker(pingEvery)
		defer t.Stop()
		for {
			select {
			case <-stopPing:
				return
			case <-t.C:
				if write(0x9, []byte("cbw")) != nil {
					conn.Close()
					return
				}
			}
		}
	}()

	var msg []byte
	for {
		conn.SetReadDeadline(time.Now().Add(readTimeout))
		fin, op, payload, err := readFrame(br)
		if err != nil {
			return err
		}
		switch op {
		case 0x0, 0x1, 0x2: // continuação, texto, binário
			msg = append(msg, payload...)
			if fin {
				fn(strings.TrimSpace(string(msg)))
				msg = msg[:0]
			}
		case 0x8: // close
			write(0x8, nil)
			return errors.New("fechada pelo Timing")
		case 0x9: // ping
			if err := write(0xA, payload); err != nil {
				return err
			}
		case 0xA: // pong do nosso ping: só renova o prazo de leitura
		}
	}
}

func readFrame(r *bufio.Reader) (fin bool, op byte, payload []byte, err error) {
	var h [2]byte
	if _, err = io.ReadFull(r, h[:]); err != nil {
		return
	}
	fin, op = h[0]&0x80 != 0, h[0]&0x0F
	masked := h[1]&0x80 != 0
	n := uint64(h[1] & 0x7F)
	switch n {
	case 126:
		var b [2]byte
		if _, err = io.ReadFull(r, b[:]); err != nil {
			return
		}
		n = uint64(binary.BigEndian.Uint16(b[:]))
	case 127:
		var b [8]byte
		if _, err = io.ReadFull(r, b[:]); err != nil {
			return
		}
		n = binary.BigEndian.Uint64(b[:])
	}
	if n > 4<<20 {
		err = fmt.Errorf("quadro grande demais: %d bytes", n)
		return
	}
	var mask [4]byte
	if masked {
		if _, err = io.ReadFull(r, mask[:]); err != nil {
			return
		}
	}
	payload = make([]byte, n)
	if _, err = io.ReadFull(r, payload); err != nil {
		return
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return
}

// writeFrame envia um quadro do cliente (sempre mascarado, exigência da RFC 6455).
func writeFrame(w io.Writer, op byte, payload []byte) error {
	if len(payload) > 125 {
		payload = payload[:125]
	}
	var mask [4]byte
	rand.Read(mask[:])
	buf := []byte{0x80 | op, 0x80 | byte(len(payload))}
	buf = append(buf, mask[:]...)
	for i, b := range payload {
		buf = append(buf, b^mask[i%4])
	}
	_, err := w.Write(buf)
	return err
}
