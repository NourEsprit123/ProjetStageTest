# --- Étape 1 : build du binaire Go ---
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Maintenant que go.mod est dans le dossier backend, 
# cette ligne fonctionnera car le Dockerfile est aussi dans backend/
COPY go.mod go.sum ./
RUN go mod download

# Copier le reste du code source (tout le dossier backend)
COPY . .

# Compiler en binaire statique
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

# --- Étape 2 : image finale légère ---
FROM alpine:latest
WORKDIR /app
RUN apk update && apk add --no-cache ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]