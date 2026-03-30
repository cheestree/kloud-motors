FROM golang:1.25-alpine AS builder
WORKDIR /app
# Copy go.mod and go.sum from the build context root
COPY go.mod go.sum ./
# Copy the shared and search service source code
COPY ./shared ./shared
COPY ./search ./search
WORKDIR /app/search
RUN go mod tidy && go build -o search-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/search/search-service .
EXPOSE 50051
CMD ["./search-service"]
