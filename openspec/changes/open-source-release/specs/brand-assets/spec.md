# Spec Delta

## Purpose

Define como logos e créditos são carregados e exibidos, para que o repositório funcione sem configuração, com a logo da CBW como placeholder, e mantenha a identidade própria do cbw-live e o crédito da Wizd Digital.

## ADDED Requirements

### Requirement: Placeholder quando a logo da organização não existe
O serviço SHALL servir `/assets/logo.svg` a partir de `ASSETS_DIR` e, se o arquivo não existir, MUST servir o placeholder incluído no repositório (a logo da CBW).

#### Scenario: Instalação limpa
- **WHEN** o cbw-live roda sem `assets/logo.svg`
- **THEN** o overlay e a home mostram a logo da CBW, sem imagem quebrada

#### Scenario: Logo própria fornecida
- **WHEN** o usuário coloca seu `assets/logo.svg`
- **THEN** o serviço serve esse arquivo no lugar do placeholder

### Requirement: Identidade do cbw-live
O projeto SHALL ter logo própria em `assets/brand/`, usada no topo dos READMEs e na home.

#### Scenario: README no GitHub
- **WHEN** alguém abre o README em tema claro ou escuro
- **THEN** a logo do cbw-live é exibida legível nos dois temas

### Requirement: Crédito da Wizd Digital
A home e os READMEs SHALL exibir "Desenvolvido por Wizd Digital" com link para `https://wizddigital.com`, com versão da logo legível em fundo claro e escuro.

#### Scenario: Clique no crédito
- **WHEN** o usuário clica no crédito
- **THEN** abre `https://wizddigital.com`
