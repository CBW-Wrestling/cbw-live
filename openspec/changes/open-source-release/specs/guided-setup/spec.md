# Spec Delta

## Purpose

Permite configurar o cbw-live pela primeira vez usando só o próprio executável, de forma idêntica em macOS, Windows e Linux, sem bash nem ferramentas externas.

## ADDED Requirements

### Requirement: Subcomando de setup interativo
O executável SHALL oferecer o subcomando `cbwlive setup`, que pergunta URL do Arena, App ID, App Secret, API Key, ID do evento e IPs dos PCs do UWW Timing e grava o arquivo `.env`.

#### Scenario: Primeira configuração
- **WHEN** o usuário roda `cbwlive setup` e não existe `.env`
- **THEN** o programa faz as perguntas e cria o `.env` com `WEBHOOK_TOKEN` aleatório e `MERCURE_HUB_URL` derivada da URL do Arena

#### Scenario: Segredos não aparecem na tela
- **WHEN** o usuário digita o App Secret ou a API Key
- **THEN** o texto digitado não é exibido no terminal

### Requirement: Não sobrescrever configuração existente sem confirmação
O setup SHALL pedir confirmação antes de substituir um `.env` existente e MUST manter o arquivo intacto se o usuário não confirmar.

#### Scenario: .env já existe
- **WHEN** o usuário roda `cbwlive setup` com um `.env` existente e responde que não quer sobrescrever
- **THEN** o programa encerra sem alterar o arquivo

### Requirement: Chave do Mercure sem Docker obrigatório
O setup SHALL tentar ler a chave do Mercure do container `caddy` do Arena quando o Docker estiver disponível e, caso contrário, MUST pedir a chave ao usuário.

#### Scenario: Docker indisponível
- **WHEN** o comando `docker` não existe ou o container `caddy` não responde
- **THEN** o setup pergunta `MERCURE_PUBLISHER_JWT_KEY` e continua

### Requirement: Proteção do arquivo de segredos
O setup SHALL restringir o `.env` ao usuário atual onde o sistema operacional permitir (permissão 600 em Unix) e MUST NOT versionar nem imprimir os segredos.

#### Scenario: Unix
- **WHEN** o `.env` é criado em macOS ou Linux
- **THEN** sua permissão é 600

#### Scenario: Windows
- **WHEN** o `.env` é criado no Windows
- **THEN** o setup conclui sem erro e avisa que o arquivo contém segredos
