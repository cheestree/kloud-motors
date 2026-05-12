FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./shared ./shared
COPY ./search ./search
COPY ./observability ./observability
COPY ./utils ./utils
COPY ./redis ./redis

WORKDIR /app/search
RUN go build -o search-service main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/search/search-service .
EXPOSE 50056
CMD ["./search-service"]
