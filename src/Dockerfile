# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Final stage
FROM alpine:3.21
WORKDIR /app

# Créer un utilisateur non-root
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Installation des dépendances minimales et AWS CLI
RUN apk add --no-cache aws-cli && \
    mkdir -p /home/appuser/.kube && \
    chown -R appuser:appgroup /home/appuser/.kube

# Copie des fichiers de l'application
COPY --from=builder /app/main .
COPY views views/
COPY static static/

# Configuration des permissions
RUN chown -R appuser:appgroup /app && \
    chmod -R 755 /app

# Passage à l'utilisateur non-root
USER appuser

EXPOSE 3000

CMD ["./main"]