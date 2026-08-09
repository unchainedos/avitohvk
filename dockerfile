FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /app/config

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/main.go

FROM ubuntu:latest

WORKDIR /

COPY --from=builder /server /server

COPY --from=builder /app/config/config.yaml /config/config.yaml

EXPOSE 8080

ENTRYPOINT ["/server"]