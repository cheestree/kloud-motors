FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./geographic-maket-insights ./geographic-maket-insights

WORKDIR /app/geographic-maket-insights
RUN go build -o geographic-maket-insights-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/geographic-maket-insights/geographic-maket-insights-service .
EXPOSE 50052
CMD ["./geographic-maket-insights-service"]
