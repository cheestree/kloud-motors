FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./geographic-market-insights ./geographic-market-insights
COPY ./utils ./utils

WORKDIR /app/geographic-market-insights
RUN go build -o geographic-market-insights-service .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/geographic-market-insights/geographic-market-insights-service .
EXPOSE 50053
CMD ["./geographic-market-insights-service"]
