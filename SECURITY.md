# Política de segurança / Security policy

**Português** · [English](#english)

## Como reportar uma vulnerabilidade

**Não abra uma issue pública** para falhas de segurança. Envie os detalhes em particular:

- E-mail: wellyngson.guedes@gmail.com
- LinkedIn (mensagem direta): https://www.linkedin.com/in/wellyngson

Inclua, se possível: versão do cbw-live, sistema operacional, passos para reproduzir e o impacto que você percebe. **Não envie** `.env`, API Keys, secrets ou chaves do Mercure.

Respondemos assim que possível, em geral em poucos dias, e combinamos com você o momento de divulgar a correção.

## Versões suportadas

Só a versão mais recente publicada em [Releases](https://github.com/CBW-Wrestling/cbw-live/releases) recebe correções.

## Limites conhecidos (não são vulnerabilidades a reportar)

O cbw-live foi feito para rodar **apenas em uma rede local de produção**. Por projeto:

- Não tem autenticação: quem acessa a porta 8095 abre a página de controle.
- O token do webhook ainda não é exigido (`WEBHOOK_TOKEN_ENFORCE=false`).
- A porta 9696 do UWW Timing e o Redis do Arena (`16370`) não têm autenticação e não são do cbw-live.

Não exponha o serviço à internet. Veja a seção "Segurança" do [README](README.md#segurança). Melhorias nesses pontos são bem-vindas como contribuição.

---

## English

### Reporting a vulnerability

**Please do not open a public issue** for security problems. Send the details privately:

- Email: wellyngson.guedes@gmail.com
- LinkedIn (direct message): https://www.linkedin.com/in/wellyngson

If possible, include the cbw-live version, operating system, steps to reproduce and the impact you see. **Do not send** `.env` files, API keys, secrets or Mercure keys.

We reply as soon as possible, usually within a few days, and agree with you on when to disclose the fix.

### Supported versions

Only the latest version published on [Releases](https://github.com/CBW-Wrestling/cbw-live/releases) receives fixes.

### Known limits (not vulnerabilities to report)

cbw-live is designed to run **only on a local production network**. By design:

- It has no authentication: anyone who can reach port 8095 can open the control page.
- The webhook token is not enforced yet (`WEBHOOK_TOKEN_ENFORCE=false`).
- UWW Timing's port 9696 and Arena's Redis (`16370`) have no authentication and are not part of cbw-live.

Do not expose the service to the internet. See the "Security" section of the [README](README.en.md#security). Improvements here are welcome as contributions.
