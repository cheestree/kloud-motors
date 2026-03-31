FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./chat ./chat

WORKDIR /app/chat
RUN go build -o chat-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/chat/chat-service .
EXPOSE 50052
CMD ["./chat-service"]
