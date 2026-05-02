FROM node:22-alpine3.21 AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS go-builder
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN test -f ./frontend/dist/index.html
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /bin/api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=go-builder /bin/api /bin/api
EXPOSE 8080
CMD ["/bin/api"]
