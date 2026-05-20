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
COPY ./geographic-market-insights/proto ./geographic-market-insights/proto
COPY ./auction/proto                 ./auction/proto
COPY ./marketprice/proto             ./marketprice/proto
COPY ./observability                 ./observability
COPY ./gateway                       ./gateway
COPY ./utils ./utils

RUN go build -o gateway/gateway ./gateway

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gateway/gateway .
EXPOSE 8080
CMD ["./gateway"]
