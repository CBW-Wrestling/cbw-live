# Proposal

## Why

O cbw-live virou um projeto open source mantido pela Wizd Digital e aprovado pela CBW. O repositório já é público, mas não tem licença (na prática, todos os direitos reservados), documentação só em português, nenhuma proteção na `main`, logo da CBW real versionada e setup que só funciona com bash. Falta o que torna o projeto utilizável e contribuível por terceiros, em macOS, Windows e Linux.

## What Changes

- Licença **Apache-2.0** (titular: Wizd Digital) e `NOTICE` deixando marcas (CBW, UWW, Wizd Digital) fora da licença.
- README em português e inglês (`README.md` e `README.en.md`, com link de troca de idioma), aviso "não afiliado à UWW" e crédito "Desenvolvido por Wizd Digital" (`https://wizddigital.com`).
- Identidade visual: logo própria do cbw-live, a logo da CBW é o **placeholder** padrão versionado no repo, logo da Wizd Digital com versão para fundo claro.
- Documentos de comunidade: `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, templates de issue e PR.
- CI em matriz Linux, Windows e macOS (`go vet`, `go test -race`) e releases com binários para os três sistemas.
- **Setup sem bash:** subcomando `cbwlive setup` que substitui `scripts/setup-env.sh` em qualquer sistema.
- Instruções de instalação por sistema operacional.
- Ajustes no GitHub: proteção da `main` (PR, 1 aprovação, CI verde, sem force-push), secret scanning, Dependabot, varredura com `gitleaks`.
- Autoria: e-mail dos commits passa a ser o do Gmail do mantenedor. **BREAKING** se incluir reescrita do histórico (hashes mudam, force-push). Essa etapa só roda com confirmação explícita no momento da execução.

## Capabilities

### New Capabilities
- `guided-setup`: configuração inicial do cbw-live (geração do `.env`) pelo próprio executável, igual em macOS, Windows e Linux, sem depender de bash.
- `brand-assets`: logo da organização e logo do cbw-live carregadas de `assets/`, com a logo da CBW como placeholder quando `assets/logo.svg` não existe, e crédito da Wizd Digital na home.

### Modified Capabilities

(nenhuma: o projeto ainda não tem specs)

## Impact

- Código: novo subcomando em `cmd/cbwlive/`, ajuste do fallback de logo no overlay (`web/`) e na home (`web/index.html`).
- Repo: `LICENSE`, `NOTICE`, `README*.md`, `CONTRIBUTING.md`, `SECURITY.md`, `CODE_OF_CONDUCT.md`, `.github/` (workflows, templates), `.goreleaser.yaml`, `assets/brand/`.
- Remoto GitHub: regras da `main` e recursos de segurança.
- `scripts/setup-env.sh` permanece como atalho opcional em Unix.
- Se houver reescrita do histórico: todos os clones e forks existentes divergem.
