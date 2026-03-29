FROM golang:1.25-alpine AS builder
WORKDIR /app
# Copy go.mod and go.sum from the build context root
COPY go.mod go.sum ./
# Copy the shared and listing service source code
COPY ./shared ./shared
COPY ./listing ./listing
WORKDIR /app/listing
RUN go mod tidy && go build -o listing-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/listing/listing-service .
EXPOSE 50051
CMD ["./listing-service"]
