package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"google.golang.org/grpc"

	"services/geographic-market-insights/proto"
	"services/geographic-market-insights/repository"
	"services/geographic-market-insights/repository/postgres"
	"services/utils"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	grpcPort := utils.GetEnv("GEO_GRPC_PORT", "50053")
	postgresDSN := utils.GetEnv("GEO_DATABASE_URL", utils.GetEnv("LISTING_DATABASE_URL", ""))
	if postgresDSN == "" {
		logger.Error("GEO_DATABASE_URL or LISTING_DATABASE_URL is required")
	}

	repoConfig := repository.DBConfig{
		Schema:       utils.GetEnv("POSTGRES_SCHEMA", "public"),
		Table:        utils.GetEnv("POSTGRES_TABLE", "automotive_data"),
		DefaultLimit: utils.GetEnvInt("DEFAULT_LIMIT", 20),
		MaxLimit:     utils.GetEnvInt("MAX_LIMIT", 100),
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
