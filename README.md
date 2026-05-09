# uy3-leads-api

API interna em Go para ingestao de webhooks da UY3 e disponibilizacao de dados de leads para o painel.

## Objetivo

Esta aplicacao centraliza o fluxo de:

- recebimento de leads via webhook
- autenticacao de usuarios internos
- resumo agregado de leads para a SPA
- exportacao de leads em CSV

## Stack

- Go 1.26
- Chi para roteamento HTTP
- Turso libSQL como banco
- JWT para autenticacao
- bcrypt para validacao de senha

## Estrutura

```text
cmd/api
internal/auth
internal/config
internal/database
internal/handlers
internal/middleware
internal/models
api
```

## Variaveis de ambiente

O projeto le o arquivo `.env` da raiz. Se a aplicacao for executada de dentro de `cmd/api`, ela tambem sobe a arvore de diretorios para encontrar esse arquivo.

Variaveis esperadas:

- `PORT`
- `DATABASE_URL`
- `UY3_WEBHOOK_SECRET`
- `JWT_SECRET`

Exemplo:

```env
PORT=8000
DATABASE_URL=libsql://seu-banco.turso.io?authToken=...
UY3_WEBHOOK_SECRET=...
JWT_SECRET=...
```

## Como rodar

Na raiz do projeto:

```bash
go run ./cmd/api
```

Ou de dentro de `cmd/api`:

```bash
go run .
```

## Endpoints

### Publicos

- `GET /health`
- `POST /login`
- `POST /webhook`

### Protegidos por JWT

- `GET /me`
- `GET /leads`
- `GET /leads/export`

## Autenticacao

O login recebe `email` e `password` e retorna um JWT com expiracao de 24 horas.

Os endpoints protegidos esperam:

```text
Authorization: Bearer <token>
```

## Webhook UY3

O endpoint `POST /webhook` valida o segredo da UY3 antes de processar o payload.

Headers aceitos:

- `X-UY3-Secret-Key`
- `X-Secret-Key`
- `Secret-Key`
- `Authorization: Bearer <secret>`

## Resumo de leads

O endpoint `GET /leads` retorna métricas agregadas para o painel e suporta:

- `period` com `all`, `24h`, `7d`, `30d`, `90d`
- `from` e `to` no formato `YYYY-MM-DD` ou `YYYY-MM-DDTHH:mm`

Se `from` e `to` forem enviados, eles sobrescrevem `period`. Datas sem hora continuam aceitas por compatibilidade. Quando a hora é enviada, o intervalo é interpretado no horário local de São Paulo e comparado com `leads.received_at` em UTC.

Exemplos:

```text
/leads?period=custom&from=2026-05-09T10:30&to=2026-05-09T12:45
/leads?period=custom&from=2026-05-09T08:00&to=2026-05-09T18:00
```

O campo `to` pode estar no futuro. Isso permite acompanhar intervalos ainda abertos, como um turno do dia atual.

Para manter baixo consumo de rows read no Turso:

- periodos fixos e ranges custom sem hora usam `leads_summary_daily`
- ranges custom com hora usam `COUNT(*)` direto em `leads` com `WHERE received_at >= ? AND received_at < ?`
- nenhuma query aplica funcao sobre `received_at`, preservando o uso do indice `(received_at, id)`
- ranges custom encerrados podem ser reaproveitados do cache local do navegador
- ranges custom abertos ou futuros nao usam cache e atualizam em polling mais lento, a cada 20 minutos

## Exportacao CSV

O endpoint `GET /leads/export` usa apenas filtros de data ou periodo. A exportacao sem filtro util, como `period=all`, e bloqueada para evitar consumo acidental do banco.

Regras atuais:

- streaming direto na resposta HTTP
- BOM UTF-8 no inicio do arquivo
- delimitador `;`
- cabecalhos compatíveis com o legado
- datas formatadas preferencialmente no padrao brasileiro

Exemplo com data e hora:

```text
/leads/export?period=custom&from=2026-05-09T10:30&to=2026-05-09T12:45
```

## Postman

A collection para testes manuais esta em:

[`api/uy3-leads-api.postman_collection.json`](https://github.com/luanlucolli/uy3-leads-api/blob/main/api/uy3-leads-api.postman_collection.json)

## Comportamento no startup

A aplicacao valida configuracao e conexao com o banco antes de subir o servidor HTTP.

Se o banco nao responder ao `Ping`, o processo encerra e a API nao abre a porta.

## Observacoes

- esta API foi pensada para uso interno
- alteracoes de banco nao sao aplicadas automaticamente; se necessario, confirme manualmente que existe o indice `idx_leads_pagination` em `leads (received_at, id)`
