# --- Étape 1 : build du binaire Go ---
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copier go.mod et go.sum d'abord pour profiter du cache Docker
COPY go.mod go.sum ./
RUN go mod download

# Copier le reste du code source
COPY . .

# Compiler en binaire statique (sans dépendance CGO)
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# --- Étape 2 : image finale légère ---
FROM alpine:latest

WORKDIR /app

# Certificats SSL nécessaires pour les requêtes HTTPS vers Tunisianet/Mytek
RUN apk --no-cache add ca-certificates

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]