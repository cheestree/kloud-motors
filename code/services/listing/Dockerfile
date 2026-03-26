FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o listing-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/listing-service .
EXPOSE 50052
CMD ["./listing-service"]
