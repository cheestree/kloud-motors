FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./auth ./auth
COPY ./user/proto ./user/proto

WORKDIR /app/auth
RUN go build -o auth-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/auth/auth-service .
EXPOSE 50057
CMD ["./auth-service"]
