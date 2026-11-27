package main

import (
	"io/fs"
	"net/http"
	"path"
)

// assetsHandler serve /assets/*: arquivo em dir (ASSETS_DIR) tem prioridade;
// se não existir, usa os arquivos embutidos no executável. Sem assets/logo.svg
// (logo própria de quem usa), serve logo.placeholder.svg (a logo da CBW).
func assetsHandler(dir string, embedded fs.FS) http.Handler {
	disk := http.FileServer(http.Dir(dir))
	diskFS := http.Dir(dir)
	emb := http.FileServer(http.FS(embedded))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := path.Clean("/" + r.URL.Path)
		if f, err := diskFS.Open(name); err == nil {
			st, serr := f.Stat()
			f.Close()
			if serr == nil && !st.IsDir() {
				disk.ServeHTTP(w, r)
				return
			}
		}
		if name == "/logo.svg" {
			name = "/logo.placeholder.svg"
		}
		r2 := r.Clone(r.Context())
		r2.URL.Path = name
		emb.ServeHTTP(w, r2)
	})
}
