package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	proto "geographic-maket-insights/proto"
	"geographic-maket-insights/repository"
	"geographic-maket-insights/repository/postgres"
)

func main() {
	grpcPort := getenv("GRPC_PORT", "50052")
	postgresDSN := getenv("POSTGRES_DSN", "")
	if postgresDSN == "" {
		log.Fatal("POSTGRES_DSN is required")
	}

	pool, err := pgxpool.New(context.Background(), postgresDSN)
	if err != nil {
		log.Fatalf("postgres connect error: %v", err)
	}
	defer pool.Close()

	serverConfig := repository.QueryConfig{
		Schema:       getenv("POSTGRES_SCHEMA", "public"),
		Table:        getenv("POSTGRES_TABLE", "listings"),
		DefaultLimit: getenvInt("DEFAULT_LIMIT", 20),
		MaxLimit:     getenvInt("MAX_LIMIT", 100),
	}

	repo := postgres.NewPostgresRepo(pool, serverConfig)

	grpcSrv := grpc.NewServer()
	proto.RegisterGeoMarketInsightsServiceServer(grpcSrv, NewGeoServer(repo))

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}

	log.Printf("geo-market-insights gRPC listening on :%s", grpcPort)
	if err := grpcSrv.Serve(lis); err != nil {
		log.Fatalf("grpc serve error: %v", err)
	}
}

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
