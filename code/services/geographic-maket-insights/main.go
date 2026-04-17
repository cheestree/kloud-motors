package main

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"

	"services/geographic-maket-insights/proto"
	"services/geographic-maket-insights/repository"
	"services/geographic-maket-insights/repository/postgres"
)

func main() {
	grpcPort := getenv("GRPC_PORT", "50052")
	postgresDSN := getenv("GEO_DATABASE_URL", "")
	if postgresDSN == "" {
		log.Fatal("GEO_DATABASE_URL is required")
	}

	repoConfig := repository.DBConfig{
		Schema:       getenv("POSTGRES_SCHEMA", "public"),
		Table:        getenv("POSTGRES_TABLE", "automotive_data"),
		DefaultLimit: getenvInt("DEFAULT_LIMIT", 20),
		MaxLimit:     getenvInt("MAX_LIMIT", 100),
		Dsn:          getenv("GEO_DATABASE_URL", "localhost"),
	}

	repo, err := postgres.NewPostgresRepo(context.Background(), repoConfig)
	if err != nil {
		log.Fatalf("postgres repo init error: %v", err)
	}
	defer repo.Close()

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
