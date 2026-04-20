FROM golang:1.24 AS proto-gen

# Install protoc manually to get well-known types bundled
ENV PROTOC_VERSION=27.0
RUN apt-get update && apt-get install -y unzip curl && \
    curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip" && \
    unzip "protoc-${PROTOC_VERSION}-linux-x86_64.zip" -d /usr/local && \
    rm "protoc-${PROTOC_VERSION}-linux-x86_64.zip"

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

ENV PATH="${PATH}:/usr/local/bin:/root/go/bin"

WORKDIR /workspace
COPY . .

RUN protoc \
    --proto_path=/workspace \
    --proto_path=/usr/local/include \
    --go_out=/workspace \
    --go-grpc_out=/workspace \
    --go_opt=paths=source_relative \
    --go-grpc_opt=paths=source_relative \
    shared/shared.proto \
    listing/proto/listing.proto \
    search/proto/search.proto \
    chat/proto/chat.proto \
    seller/proto/seller.proto \
    user/proto/user.proto \
    auth/proto/auth.proto \
    geographic-maket-insights/proto/geo-market-insights.proto \
    auction/proto/auction.proto \
    marketprice/proto/marketprice.proto