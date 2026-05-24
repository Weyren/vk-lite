FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/vk-lite ./cmd/vk-lite

FROM alpine:3.22

WORKDIR /app

COPY --from=build /out/vk-lite /app/vk-lite

EXPOSE 8080
ENTRYPOINT ["/app/vk-lite"]
