FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o seller-service main.go listing.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/seller-service .
EXPOSE 50054
CMD ["./seller-service"]
