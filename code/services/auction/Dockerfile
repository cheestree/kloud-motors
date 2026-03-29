FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

# Download dependencies
RUN go mod tidy

# Build the app
RUN go build -o auction-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/auction-service .

EXPOSE 50054
EXPOSE 8080

CMD ["./auction-service"]
