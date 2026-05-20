package main

import (
	"context"
	"log/slog"
	"os"

	"google.golang.org/grpc"

	geopb "services/geographic-market-insights/proto"
	"services/geographic-market-insights/repository"
	"services/geographic-market-insights/repository/postgres"
	"services/observability"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"services/utils"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	ctx := context.Background()
	shutdownTracing := observability.InitTracing(ctx, logger, "geographic-market-insights")
	defer func() {
		if err := shutdownTracing(ctx); err != nil {
			logger.Error("failed to shutdown tracing", "error", err)
		}
	}()

	geoGrpcPort := utils.MustGetEnv("GEO_GRPC_PORT")
	geoDSN := utils.GetEnv("GEO_DATABASE_URL", utils.GetEnv("LISTING_DATABASE_URL", ""))
	if geoDSN == "" {
		logger.Error("GEO_DATABASE_URL or LISTING_DATABASE_URL is required")
	}

	repoConfig := repository.DBConfig{
		Schema:       utils.GetEnv("POSTGRES_SCHEMA", "public"),
		Table:        utils.GetEnv("POSTGRES_TABLE", "automotive_data"),
		DefaultLimit: utils.GetEnvInt("DEFAULT_LIMIT", 20),
		MaxLimit:     utils.GetEnvInt("MAX_LIMIT", 100),
		Dsn:          geoDSN,
	}

	repo, err := postgres.NewPostgresRepo(ctx, repoConfig)
	if err != nil {
		logger.Error("postgres repo init error", "error", err)
		return
	}
	defer repo.Close()

	lis := utils.TryListen(geoGrpcPort)

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	geopb.RegisterGeoMarketInsightsServiceServer(grpcServer, NewGeoServer(repo))

	utils.HealthCheck("geographic-market-insights.GeoMarketInsightsService", grpcServer)

	logger.Info("Geo Market Insights gRPC server is running", "addr", lis.Addr().String())

	utils.TryServe(grpcServer, lis)
}
