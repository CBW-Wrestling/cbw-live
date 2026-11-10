package timing

import (
	"regexp"
	"strconv"
	"strings"
)

// Msg é uma mensagem do Timing já separada: Head é o código ("clk", "ch1", …)
// ou "state" para a linha de estado completo; Args são os campos seguintes,
// sem os tokens chave=valor (applyAt=…).
type Msg struct {
	Head  string
	Args  []string
	State *State
}

// State é a linha de estado completo que o Timing envia ao conectar
// (campos medidos em 10/11/2026; a página oficial usa os mesmos índices).
type State struct {
	MatchNo       string // [1]
	Category      string // [2] "Seniors FS 74"
	Audience      string // [3]
	Clock         string // [4]
	HomeCountry   string // [5]
	HomeName      string // [6]
	AwayCountry   string // [7]
	AwayName      string // [8]
	HomeScore     int    // [9]
	AwayScore     int    // [10]
	HomeCautions  int    // [13]
	AwayCautions  int    // [14]
	Advantage     int    // [19] 0 nenhum, 1 vermelho, 2 azul
	Winner        int    // [20] -1/0 nenhum, 1 vermelho, 2 azul
	Victory       string // [21]
	Round         string // [22] "Qualif."
	HomeColor     string // [23]
	AwayColor     string // [24]
	RoundDuration int    // [25] segundos
	Rounds        int    // [26]
}

// Parse interpreta uma mensagem crua do WebSocket.
func Parse(raw string) Msg {
	raw = strings.TrimSpace(raw)
	if rest, ok := strings.CutPrefix(raw, "evt;"); ok {
		parts := strings.Split(rest, ";")
		args := make([]string, 0, len(parts))
		for _, p := range parts[1:] {
			p = strings.TrimSpace(p)
			if strings.Contains(p, "=") {
				continue // applyAt=…
			}
			args = append(args, p)
		}
		return Msg{Head: strings.ToLower(strings.TrimSpace(parts[0])), Args: args}
	}
	f := strings.Split(raw, ";")
	if len(f) < 25 {
		return Msg{Head: "unknown", Args: f}
	}
	at := func(i int) string {
		if i < len(f) {
			return strings.TrimSpace(f[i])
		}
		return ""
	}
	n := func(i int) int { v, _ := strconv.Atoi(at(i)); return v }
	return Msg{Head: "state", State: &State{
		MatchNo: at(1), Category: at(2), Audience: at(3), Clock: at(4),
		HomeCountry: at(5), HomeName: at(6), AwayCountry: at(7), AwayName: at(8),
		HomeScore: n(9), AwayScore: n(10), HomeCautions: n(13), AwayCautions: n(14),
		Advantage: n(19), Winner: n(20), Victory: at(21), Round: at(22),
		HomeColor: at(23), AwayColor: at(24), RoundDuration: n(25), Rounds: n(26),
	}}
}

// Arg devolve o argumento i ("" se não existir).
func (m Msg) Arg(i int) string {
	if i < len(m.Args) {
		return m.Args[i]
	}
	return ""
}

// Pair lê mensagens no formato "sc1;X;sc2;Y" / "ct1;X;ct2;Y": devolve X e Y.
func (m Msg) Pair() (int, int, bool) {
	x, err1 := strconv.Atoi(m.Arg(0))
	y, err2 := strconv.Atoi(m.Arg(2))
	return x, y, err1 == nil && err2 == nil
}

// ChallengeStatus traduz "ch1;a;b": ""=pedido, "1"=aceito/em análise,
// "1;1"=ganhou, "1;0"=perdeu, "0"=cancelado.
func (m Msg) ChallengeStatus() string {
	a, b := m.Arg(0), m.Arg(1)
	switch {
	case a == "":
		return "requested"
	case a == "0":
		return "canceled"
	case a == "1" && b == "":
		return "challenge"
	case a == "1" && b == "1":
		return "won"
	case a == "1" && b == "0":
		return "lost"
	}
	return ""
}

// Side devolve "home" para códigos terminados em 1 (vermelho) e "away" para 2 (azul).
func (m Msg) Side() string {
	switch {
	case strings.HasSuffix(m.Head, "1"):
		return "home"
	case strings.HasSuffix(m.Head, "2"):
		return "away"
	}
	return ""
}

var nameCountry = regexp.MustCompile(`^(.*?)\s*\(([A-Za-z]{2,4})\)\s*$`)

// SplitNameCountry separa "GOZIUMOV K.(AZE)" em ("GOZIUMOV K.", "AZE").
func SplitNameCountry(s string) (string, string) {
	if m := nameCountry.FindStringSubmatch(s); m != nil {
		return m[1], strings.ToUpper(m[2])
	}
	return strings.TrimSpace(s), ""
}

var victoryCode = regexp.MustCompile(`([A-Z]{2,}[0-9]*)\s*$`)

// VictoryCode extrai o código de "5-0VFA" → "VFA".
func VictoryCode(s string) string {
	if m := victoryCode.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(s))); m != nil {
		return m[1]
	}
	return ""
}
