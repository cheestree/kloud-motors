FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o user-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/user-service .
EXPOSE 50053
CMD ["./user-service"]
