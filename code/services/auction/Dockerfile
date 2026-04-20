FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./shared ./shared
COPY ./auction ./auction
COPY ./listing ./listing

WORKDIR /app/auction
RUN go build -o auction-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/auction/auction-service .

EXPOSE 50051
EXPOSE 8080

CMD ["./auction-service"]
