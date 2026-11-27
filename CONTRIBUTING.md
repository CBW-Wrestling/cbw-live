# Contribuindo / Contributing

**Português** · [English](#english)

Obrigado por querer ajudar! Este projeto é mantido pela Wizd Digital e criado para a CBW.

## Antes de começar

- Dúvidas, bugs e ideias: abra uma [issue](https://github.com/CBW-Wrestling/cbw-live/issues) usando os modelos.
- Mudanças grandes (novo gráfico, novo protocolo, mudança de arquitetura): abra uma issue antes, para alinharmos o escopo.
- Vulnerabilidades: não use issues. Veja [SECURITY.md](SECURITY.md).
- Ao participar, você concorda com o [Código de Conduta](CODE_OF_CONDUCT.md).

## Ambiente

Precisa do Go (versão do `go.mod`). Funciona em macOS, Windows e Linux.

```bash
go vet ./...
go test -race ./...
go run ./cmd/cbwlive -debug     # precisa de um .env: ./cbwlive setup
```

As páginas em `web/` e os arquivos em `assets/` (placeholder, marcas) são embutidos no executável: recompile após editar. Para testar o visual sem Arena nem Timing, use a galeria em `/preview.html`.

## Pull requests

1. Faça um fork e crie uma branch a partir da `main`.
2. Mantenha o PR pequeno e focado em uma coisa.
3. Rode `gofmt`, `go vet ./...` e `go test -race ./...`. O CI roda os mesmos testes em Linux, Windows e macOS.
4. Inclua testes quando mudar comportamento (veja `internal/live/testdata` para mensagens reais do Timing).
5. Atualize `README.md` **e** `README.en.md` se a mudança afetar o uso.
6. A `main` é protegida: toda mudança entra por PR, com aprovação do mantenedor e CI verde.

## O que nunca deve entrar em um PR

- `.env`, API Keys, secrets, chaves do Mercure, capturas com credenciais.
- Logos ou marcas de terceiros (a única exceção é o placeholder da CBW). Veja [NOTICE](NOTICE).
- Dados pessoais de atletas ou de operadores que não sejam públicos.

## Licença

Ao enviar uma contribuição, você concorda que ela será licenciada sob a [Apache-2.0](LICENSE), a mesma do projeto.

---

## English

Thanks for wanting to help! This project is maintained by Wizd Digital and was created for the CBW.

### Before you start

- Questions, bugs and ideas: open an [issue](https://github.com/CBW-Wrestling/cbw-live/issues) using the templates.
- Larger changes (new graphic, new protocol, architecture change): open an issue first so we can agree on scope.
- Vulnerabilities: do not use issues. See [SECURITY.md](SECURITY.md).
- By participating, you agree to the [Code of Conduct](CODE_OF_CONDUCT.md).

### Setup

You need Go (the version in `go.mod`). It works on macOS, Windows and Linux.

```bash
go vet ./...
go test -race ./...
go run ./cmd/cbwlive -debug     # needs a .env: ./cbwlive setup
```

The pages in `web/` and the files in `assets/` (placeholder, brand files) are embedded in the executable: rebuild after editing. To test the look without Arena or Timing, use the gallery at `/preview.html`.

### Pull requests

1. Fork and create a branch from `main`.
2. Keep the PR small and focused on one thing.
3. Run `gofmt`, `go vet ./...` and `go test -race ./...`. CI runs the same tests on Linux, Windows and macOS.
4. Add tests when you change behavior (see `internal/live/testdata` for real Timing messages).
5. Update `README.md` **and** `README.en.md` if the change affects usage.
6. `main` is protected: every change goes through a PR, with maintainer approval and green CI.

### What must never go into a PR

- `.env`, API keys, secrets, Mercure keys, screenshots showing credentials.
- Third-party logos or marks (the only exception is the CBW placeholder). See [NOTICE](NOTICE).
- Personal data of athletes or operators that is not public.

### License

By submitting a contribution, you agree that it is licensed under [Apache-2.0](LICENSE), the same as the project.
