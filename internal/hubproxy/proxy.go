// Package hubproxy expõe o hub Mercure para os overlays na mesma origem do
// serviço (sem CORS) e injeta o JWT de assinante no servidor — o navegador
// nunca vê a chave. Só permite assinar tópicos da CBW.
package hubproxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const allowedPrefix = "cbw/"

func New(hubURL, subscriberToken string) (http.Handler, error) {
	target, err := url.Parse(hubURL)
	if err != nil {
		return nil, err
	}
	rp := &httputil.ReverseProxy{
		FlushInterval: -1, // repassa cada evento SSE imediatamente
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.Out.URL.Scheme = target.Scheme
			pr.Out.URL.Host = target.Host
			pr.Out.URL.Path = target.Path
			pr.Out.Host = target.Host
			pr.Out.Header.Set("Authorization", "Bearer "+subscriberToken)
			pr.Out.Header.Del("Cookie")
		},
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "somente GET", http.StatusMethodNotAllowed)
			return
		}
		topics := r.URL.Query()["topic"]
		if len(topics) == 0 {
			http.Error(w, "informe ?topic=", http.StatusBadRequest)
			return
		}
		for _, t := range topics {
			if !strings.HasPrefix(t, allowedPrefix) {
				http.Error(w, "tópico não permitido", http.StatusForbidden)
				return
			}
		}
		rp.ServeHTTP(w, r)
	}), nil
}
