# Build stage
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 go build -o main .

# Final stage
FROM alpine:3.19
WORKDIR /app

# Installation des dépendances minimales et AWS CLI
RUN apk add --no-cache \
    aws-cli \
    && mkdir -p /root/.kube

# Copie des fichiers de l'application
COPY --from=builder /app/main .
COPY views views
COPY static static

EXPOSE 3000
CMD ["./main"]