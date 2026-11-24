# Design

## Context

Veja proposal.md (Why). Estado atual: Go único, `web/` embutido via `go:embed`, `assets/` lido do disco (`ASSETS_DIR`) e servido em `/assets/`. `web/ui.js` e `web/themes.js` referenciam `/assets/logo.svg` com `onerror` que esconde a imagem. O setup é `scripts/setup-env.sh` (bash, `openssl`, `docker compose`). Repo já público em `CBW-Wrestling/cbw-live`, sem LICENSE, `main` sem proteção.

## Goals / Non-Goals

**Goals:**
- Mesmo fluxo de instalação e setup em macOS, Windows e Linux.
- Repo público sem a marca real da CBW daqui em diante.
- Documentação bilíngue e governança mínima de open source.

**Non-Goals:**
- Instalador gráfico (.msi) ou app desktop (fica para depois).
- Autenticação no cbw-live (continua só para rede local).
- Remover a logo real do histórico git.

## Decisions

1. **Licença Apache-2.0 + NOTICE.** Cláusula de patente e de marcas. Alternativa MIT: mais curta, sem proteção de marca. Titular: Wizd Digital.
2. **Setup em Go (`cbwlive setup`).** Mesmo binário nos 3 SOs; `crypto/rand` no lugar de `openssl`; leitura oculta de segredo com `golang.org/x/term`; Docker opcional via `os/exec`, com pergunta manual como fallback. Alternativa: manter bash e adicionar `.ps1` (duas bases para manter). `setup-env.sh` fica como atalho Unix, sem evoluir.
3. **Placeholder no servidor, não no navegador.** O handler de `/assets/` usa o arquivo de `ASSETS_DIR` e, se faltar, os arquivos embutidos no executável (`assets/embed.go`: placeholder, marca cbw-live, Wizd); `logo.svg` ausente vira `logo.placeholder.svg`. Embutir garante que o binário baixado das Releases funcione sem a pasta `assets/`. Assim a home, o overlay e as 13 telas de tema funcionam sem mexer em `themes.js`. Alternativa: `onerror` trocar `src` no front (espalha lógica em vários temas).
4. **Logo real local.** `assets/logo.svg` entra no `.gitignore` e sai do índice (`git rm --cached`). Fica no histórico por decisão do mantenedor.
5. **Marca cbw-live em `assets/brand/`** (SVG claro/escuro) e Wizd com variante para fundo claro. `README` usa `<picture>` com `prefers-color-scheme`.
6. **READMEs separados** `README.md` (PT) e `README.en.md` (EN), links cruzados no topo. Alternativa: um arquivo só, longo demais.
7. **CI e release.** GitHub Actions: matriz `ubuntu/windows/macos` com `go vet` e `go test -race ./...`; tag `v*` aciona goreleaser (linux amd64/arm64, windows amd64, darwin amd64/arm64). Como o CI valida os 3 SOs, o README declara "testado em macOS; Windows/Linux validados por CI".
8. **Proteção da `main` por ruleset**: PR obrigatório, 1 aprovação, status checks do CI, sem force-push e sem delete.
9. **Autoria dos commits.** Daqui em diante `git config user.email` com o Gmail do mantenedor. Reescrever o histórico para trocar o e-mail antigo é etapa separada, só com confirmação explícita na hora.

## Risks / Trade-offs

- [Reescrita de histórico quebra clones/forks e invalida hashes] → fazer por último, antes de ativar a proteção da `main`, com backup do repo (`git clone --mirror`) e confirmação explícita.
- [E-mail novo não verificado no GitHub desvincula commits do perfil] → verificar o e-mail na conta antes.
- [Logo real continua acessível no histórico] → aceito; nota de marca no `NOTICE`.
- [Windows sem teste manual] → CI cobre build e testes; pedir feedback no README.
- [Temas "UWW oficial" imitam identidade da UWW] → mantidos por decisão do mantenedor; README e `NOTICE` declaram que o projeto não é afiliado à UWW.
- [Webhook sem token obrigatório] → documentado no `SECURITY.md`.

## Migration Plan

1. Entregar tudo em branch/PR, com a `main` ainda desprotegida.
2. Revisar e mesclar.
3. Verificar e-mail no GitHub, backup mirror, reescrita opcional do histórico, force-push.
4. Ativar a proteção da `main` e os recursos de segurança.
5. Criar a tag `v0.1.0` e conferir os binários.

Rollback da reescrita: restaurar do mirror. As demais etapas são reversíveis por revert.

## Open Questions

- Handle e e-mail de contato para `SECURITY.md` e `CODE_OF_CONDUCT.md` (pode ser definido na implementação).
