# uy3-leads-api

API interna em Go para ingestao de webhooks da UY3 e disponibilizacao de dados de leads para o painel.

## Objetivo

Esta aplicacao centraliza o fluxo de:

- recebimento de leads via webhook
- autenticacao de usuarios internos
- listagem paginada de leads para a SPA
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

O payload bruto recebido tambem e salvo no banco em `raw_payload`.

## Listagem de leads

O endpoint `GET /leads` suporta:

- `page`
- `per_page`
- `period` com `all`, `24h`, `7d`, `30d`, `90d`
- `from` e `to` no formato `YYYY-MM-DD`
- `sort` com `received_at` ou `id`
- `direction` com `asc` ou `desc`

Se `from` e `to` forem enviados, eles sobrescrevem `period`.

## Exportacao CSV

O endpoint `GET /leads/export` usa os mesmos filtros do `GET /leads`.

Regras atuais:

- streaming direto na resposta HTTP
- BOM UTF-8 no inicio do arquivo
- delimitador `;`
- cabecalhos compatíveis com o legado
- datas formatadas preferencialmente no padrao brasileiro

## Postman

A collection para testes manuais esta em:

[`api/uy3-leads-api.postman_collection.json`](https://github.com/luanlucolli/uy3-leads-api/blob/main/api/uy3-leads-api.postman_collection.json)

## Comportamento no startup

A aplicacao valida configuracao e conexao com o banco antes de subir o servidor HTTP.

Se o banco nao responder ao `Ping`, o processo encerra e a API nao abre a porta.

## Observacoes

- esta API foi pensada para uso interno

