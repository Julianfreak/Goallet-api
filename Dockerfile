# ETAPA 1: Compilación (Builder)
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copiamos archivos de dependencias
COPY go.mod ./
# COPY go.sum ./

RUN go mod download

# Copiamos el resto del código
COPY . .

# Compilamos el binario de forma estática
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/goallet-api ./cmd/api/main.go

# ETAPA 2: Entorno Limpio de Producción
FROM alpine:latest

WORKDIR /app

# Copiamos solo el binario ejecutable
COPY --from=builder /app/bin/goallet-api .

# Comando de ejecución
CMD ["./goallet-api"]