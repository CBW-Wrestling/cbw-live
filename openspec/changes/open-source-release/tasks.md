# Tasks

## 1. Licença e avisos legais

- [x] 1.1 Criar `LICENSE` (Apache-2.0, titular Wizd Digital). Verificar: `head -3 LICENSE` mostra "Apache License Version 2.0" e o ano/titular corretos.
- [x] 1.2 Criar `NOTICE` com direitos autorais da Wizd Digital e aviso de que CBW, UWW e Wizd Digital são marcas de seus titulares, fora da licença, e que o projeto não é afiliado à UWW. Verificar: arquivo existe e cita as três marcas.

## 2. Identidade visual (brand-assets)

- [x] 2.1 Criar a logo do cbw-live em `assets/brand/` (versões clara e escura, SVG). Verificar: abrir os dois SVGs no navegador sobre fundo claro e escuro, ambos legíveis.
- [x] 2.2 Criar `assets/logo.placeholder.svg` genérico (sem a marca CBW). Verificar: arquivo existe e não contém a arte da logo real.
- [x] 2.3 Criar variante da logo da Wizd Digital para fundo claro em `assets/brand/`. Verificar: visível em tema claro do GitHub.
- [x] 2.4 Servidor: `/assets/logo.svg` cai para `logo.placeholder.svg` quando o arquivo falta (`cmd/cbwlive/main.go`) e há teste. Verificar: `go test ./cmd/...` passa e `curl -I /assets/logo.svg` responde 200 sem `assets/logo.svg`.
- [x] 2.5 Tirar `assets/logo.svg` do git (`git rm --cached`), adicionar ao `.gitignore`, manter cópia local. Verificar: `git status` limpo com o arquivo presente e `git ls-files assets/logo.svg` vazio.
- [x] 2.6 Atualizar a home (`web/index.html`) com crédito "Desenvolvido por Wizd Digital" ligado a `https://wizddigital.com` e logo cbw-live. Verificar: abrir `/` e clicar no crédito.

## 3. Documentação

- [x] 3.1 Traduzir o README para inglês em `README.en.md` com todas as seções. Verificar: mesmos títulos e tabelas do `README.md`, sem trechos em português.
- [x] 3.2 Atualizar `README.md` e `README.en.md`: logo no topo, troca de idioma, aviso "não afiliado à UWW", crédito Wizd, seção de licença. Verificar: links funcionam na prévia do GitHub (claro e escuro).
- [x] 3.3 Seção de instalação por sistema (macOS, Windows, Linux), incluindo `host.docker.internal`, firewall do Windows e uso de `cbwlive setup`. Verificar: cada SO tem passos de ponta a ponta e a nota "Windows/Linux validados por CI".
- [x] 3.4 Criar `CONTRIBUTING.md`, `SECURITY.md` e `CODE_OF_CONDUCT.md`. **Antes de escrever, perguntar ao mantenedor qual contato público usar** (não usar o e-mail pessoal sem confirmação). Verificar: os três existem e linkados nos READMEs.
- [x] 3.5 Criar templates em `.github/` (bug, feature, pull request). Verificar: aparecem na tela de nova issue do GitHub.

## 4. Setup multiplataforma (guided-setup)

- [x] 4.1 Implementar `cbwlive setup` (perguntas, segredo oculto, `crypto/rand`, escrita do `.env`, confirmação ao sobrescrever). Verificar: testes de unidade cobrem criação, recusa de sobrescrita e token aleatório; `go test -race ./...` passa.
- [x] 4.2 Leitura da chave do Mercure via Docker com fallback manual. Verificar: teste com Docker simulado ausente cai na pergunta manual.
- [x] 4.3 Permissão 600 em Unix e aviso no Windows. Verificar: teste em Unix confere `0600`; build `GOOS=windows` compila.
- [x] 4.4 Manter `scripts/setup-env.sh` como atalho e apontar o README para `cbwlive setup`. Verificar: README cita os dois.

## 5. CI e release

- [x] 5.1 Workflow de CI em matriz `ubuntu`, `windows`, `macos` com `go vet` e `go test -race ./...`. Verificar: os três jobs verdes num PR.
- [ ] 5.2 `.goreleaser.yaml` e workflow por tag `v*` (linux amd64/arm64, windows amd64, darwin amd64/arm64, `web/` já embutido). Verificar: `goreleaser release --snapshot --clean` gera os binários.
- [x] 5.3 Dependabot para `gomod` e GitHub Actions. Verificar: `.github/dependabot.yml` válido.

## 6. Verificação antes de publicar

- [ ] 6.1 Rodar `gitleaks` no histórico completo. Verificar: zero achados, ou achados avaliados e rotacionados.
- [x] 6.2 Conferir se os fixtures em `testdata/` só têm dados públicos de evento de teste. Verificar: revisão manual registrada.
- [x] 6.3 Rodar tudo localmente: `go vet ./... && go test -race ./...` e build para `GOOS=windows` e `GOOS=linux`. Verificar: todos passam.

## 7. GitHub e autoria (exigem confirmação explícita)

- [x] 7.1 Configurar `git config user.email` do repo para o Gmail do mantenedor. Verificar: `git config user.email` mostra o valor.
- [x] 7.2 Confirmar com o mantenedor que o e-mail está verificado na conta GitHub e fazer backup `git clone --mirror`. Verificar: mirror existe e cobre todas as refs.
- [x] 7.3 **Somente com confirmação explícita no momento:** reescrever o histórico trocando o e-mail do autor antigo (`git filter-repo --mailmap`) e `git push --force`. Verificar: `git log --format=%ae | sort -u` mostra só o Gmail.
- [ ] 7.4 Ativar proteção da `main` (PR, 1 aprovação, CI verde, sem force-push), secret scanning e security advisories. Verificar: `gh api repos/CBW-Wrestling/cbw-live/branches/main/protection` retorna as regras.
- [ ] 7.5 Criar a tag `v0.1.0` e conferir os binários na página de releases. Verificar: assets de linux, windows e macOS presentes.
