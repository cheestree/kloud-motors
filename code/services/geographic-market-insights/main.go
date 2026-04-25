package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"strconv"

	"google.golang.org/grpc"

	"services/geographic-market-insights/proto"
	"services/geographic-market-insights/repository"
	"services/geographic-market-insights/repository/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	grpcPort := getenv("GEO_GRPC_PORT", "50053")
	postgresDSN := getenv("GEO_DATABASE_URL", getenv("LISTING_DATABASE_URL", ""))
	if postgresDSN == "" {
		logger.Error("GEO_DATABASE_URL or LISTING_DATABASE_URL is required")
	}

	repoConfig := repository.DBConfig{
		Schema:       getenv("POSTGRES_SCHEMA", "public"),
		Table:        getenv("POSTGRES_TABLE", "automotive_data"),
		DefaultLimit: getenvInt("DEFAULT_LIMIT", 20),
		MaxLimit:     getenvInt("MAX_LIMIT", 100),
		Dsn:          postgresDSN,
	}

	repo, err := postgres.NewPostgresRepo(context.Background(), repoConfig)
	if err != nil {
		logger.Error("postgres repo init error", "error", err)
	}
	defer repo.Close()

	grpcSrv := grpc.NewServer()
	proto.RegisterGeoMarketInsightsServiceServer(grpcSrv, NewGeoServer(repo))

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logger.Error("fail to listen error", "error", err)
	}

	logger.Info("geo-market-insights gRPC listening", "port", grpcPort)
	if err := grpcSrv.Serve(lis); err != nil {
		logger.Error("fail to serve error", "error", err)
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
