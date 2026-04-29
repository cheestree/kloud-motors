FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./shared ./shared
COPY ./marketprice ./marketprice
COPY ./utils ./utils

WORKDIR /app/marketprice
RUN go build -o marketprice-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/marketprice/marketprice-service .
EXPOSE 50055
CMD ["./marketprice-service"]
