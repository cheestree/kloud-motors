FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./shared ./shared
COPY ./search ./search

WORKDIR /app/search
RUN go build -o search-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/search/search-service .
EXPOSE 50051
CMD ["./search-service"]