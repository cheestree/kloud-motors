FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

# Download dependencies
RUN go mod tidy

# Build the app
RUN go build -o marketprice-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/marketprice-service .

EXPOSE 50055

CMD ["./marketprice-service"]
