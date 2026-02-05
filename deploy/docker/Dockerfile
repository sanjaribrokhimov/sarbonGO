# Этап сборки: Go 1.23 (требуется rogpeppe/go-internal и др.), загрузка зависимостей, сборка бинарника.
FROM golang:1.23-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
ENV GOPROXY=https://proxy.golang.org,direct
RUN go mod download

COPY . .
# Статическая сборка без CGO, уменьшение размера (-s -w).
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# Этап рантайма: минимальный образ Alpine, только бинарник и сертификаты.
FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /server .
EXPOSE 8080
ENTRYPOINT ["/app/server"]
