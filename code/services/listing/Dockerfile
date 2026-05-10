FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./shared ./shared
COPY ./listing ./listing
COPY ./utils ./utils
COPY ./redis ./redis

WORKDIR /app/listing
RUN go build -o listing-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/listing/listing-service .
EXPOSE 50054
CMD ["./listing-service"]
