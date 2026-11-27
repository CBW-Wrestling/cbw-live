// Package assets embute os arquivos visuais que o repositório público distribui
// (placeholder de logo, marca cbw-live, crédito Wizd Digital). Ficam no executável
// para funcionar sem a pasta assets/. assets/logo.svg (logo própria de quem usa) não é embutida.
package assets

import "embed"

//go:embed logo.placeholder.svg wizd-digital.svg brand
var Files embed.FS
