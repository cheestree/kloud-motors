FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o search-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/search-service .
EXPOSE 50051
CMD ["./search-service"]
