# uy3-leads-api

API interna em Go + React para receber webhooks da UY3, autenticar usuários, exibir métricas de leads e exportar CSV.

## Stack

- Go 1.26
- React + Vite + TypeScript
- Chi
- Turso/libSQL
- JWT + bcrypt

## Variáveis

Crie um `.env` na raiz:

```env
PORT=8080
DATABASE_URL=libsql://seu-banco.turso.io?authToken=...
UY3_WEBHOOK_SECRET=...
JWT_SECRET=...
VITE_COMPANY_NAME=
```

`PORT` é opcional. O padrão é `8080`.

## Rodar local

```bash
cd frontend && npm ci && npm run build
cd ..
go run ./cmd/api
```

## Docker

```bash
docker build -t uy3-leads-api .
docker run --env-file .env -p 8080:8080 uy3-leads-api
```

## Endpoints

Públicos:

```text
GET  /health
GET  /ready
POST /login
POST /webhook
```

Protegidos por JWT:

```text
GET /me
GET /leads
GET /leads/export
```