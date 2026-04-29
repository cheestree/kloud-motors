FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./seller ./seller
COPY ./utils ./utils

WORKDIR /app/seller
RUN go build -o seller-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/seller/seller-service .
EXPOSE 50057
CMD ["./seller-service"]
