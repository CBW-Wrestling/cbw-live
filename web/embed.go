// Package web contém as páginas estáticas (índice e overlay), embutidas no binário.
package web

import "embed"

//go:embed index.html overlay.html overlay.js overlay.css control.html settings.html preview.html mocks.js ui.css ui.js
var Files embed.FS
