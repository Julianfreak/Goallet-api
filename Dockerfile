# ETAPA 1: Compilación (Builder)
FROM golang:1.26-alpine AS builder

# Instalar ca-certificates para dependencias HTTPS durante la descarga
RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/goallet-api ./cmd/api

# ETAPA 2: Entorno Limpio de Producción
FROM alpine:3.20 AS runner

RUN apk --no-cache add ca-certificates tzdata

# Crear grupo y usuario con UID/GID explícitos no privilegiados
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

WORKDIR /app

# Copiar el binario explícitamente al directorio actual
COPY --chown=appuser:appgroup --from=builder /app/goallet-api ./goallet-api

# Cambiar al usuario sin privilegios de root
USER appuser:appgroup

EXPOSE 8080 2112

ENTRYPOINT ["./goallet-api"]