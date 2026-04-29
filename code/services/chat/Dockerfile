FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./shared ./shared
COPY ./chat ./chat
COPY ./listing/proto ./listing/proto
COPY ./seller/proto ./seller/proto
COPY ./utils ./utils

WORKDIR /app/chat
RUN go build -o chat-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/chat/chat-service .
EXPOSE 50052
CMD ["./chat-service"]
