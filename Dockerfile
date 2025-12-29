# Build Stage
FROM golang:1.20-alpine AS builder

WORKDIR /app

# Copiar arquivos de dependência
COPY go.mod ./
# Se tiver go.sum, descomente a linha abaixo
# COPY go.sum ./

RUN go mod download

# Copiar código fonte
COPY . .

# Compilar a aplicação
RUN go build -o dca-app main.go

# Run Stage
FROM alpine:latest

WORKDIR /app

# Instalar ca-certificates para chamadas HTTPS (Yahoo Finance)
RUN apk --no-cache add ca-certificates

# Copiar executável do estágio de build
COPY --from=builder /app/dca-app .

# Copiar pastas de templates e estáticos
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Expor a porta 8080
EXPOSE 8080

# Comando de execução
CMD ["./dca-app"]
