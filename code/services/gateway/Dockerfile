# Gateway service for REST to gRPC
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./shared                        ./shared
COPY ./listing/proto                 ./listing/proto
COPY ./search/proto                  ./search/proto
COPY ./seller/proto                  ./seller/proto
COPY ./user/proto                    ./user/proto
COPY ./chat/proto                    ./chat/proto
COPY ./geographic-maket-insights/proto ./geographic-maket-insights/proto
COPY ./gateway                       ./gateway

RUN go build -o gateway/gateway ./gateway

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gateway/gateway .
EXPOSE 8080
CMD ["./gateway"]
