# ETAPA 1: Compilación (Builder)
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/goallet-api ./cmd/api/main.go

# ETAPA 2: Entorno Limpio de Producción
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/bin/goallet-api .

EXPOSE 8080

CMD ["./goallet-api"]