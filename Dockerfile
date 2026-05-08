# ---------- build ----------
FROM golang:1.26.1-alpine AS builder
WORKDIR /src

# кэшируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# копируем код и собираем
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/vk-lite ./cmd/vk-lite

# ---------- runtime ----------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/vk-lite .
EXPOSE 8080
ENTRYPOINT ["./vk-lite"]
