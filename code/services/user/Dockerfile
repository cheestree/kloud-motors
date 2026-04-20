FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./user ./user

WORKDIR /app/user
RUN go build -o user-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/user/user-service .
EXPOSE 50058
CMD ["./user-service"]
