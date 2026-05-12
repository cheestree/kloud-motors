package main

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"time"

	"services/observability"
	"services/redis/cache"
	"services/search/domain"
	"services/search/proto"
	"services/search/repository"
	"services/search/service"
	"services/shared"
	"services/utils"

	_ "github.com/lib/pq"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	proto.SearchServiceServer
	service *service.SearchService
}

func (s *server) Search(ctx context.Context, req *proto.SearchRequest) (*proto.SearchResponse, error) {
	var isNew *bool
	if req.IsNew != nil {
		v := req.IsNew.Value
		isNew = &v
	}

	includeSold := false
	if req.IncludeSold != nil {
		includeSold = req.IncludeSold.Value
	}

	result, err := s.service.Search(ctx, domain.SearchParams{
		Make:         req.Make,
		Model:        req.Model,
		Year:         req.Year,
		MinPrice:     req.MinPrice,
		MaxPrice:     req.MaxPrice,
		MaxMileage:   req.MaxMileage,
		FuelType:     req.FuelType,
		BodyClass:    req.BodyClass,
		DriveType:    req.DriveType,
		Transmission: req.Transmission,
		IsNew:        isNew,
		Page:         req.Page,
		PageSize:     req.PageSize,
		IncludeSold:  includeSold,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	return toSearchResponse(result), nil
}

func toSearchResponse(result *domain.SearchResult) *proto.SearchResponse {
	listings := make([]*shared.ListingSummary, 0, len(result.Listings))
	for _, item := range result.Listings {
		listings = append(listings, toListingSummary(item))
	}

	return &proto.SearchResponse{
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
		Listings: listings,
	}
}

func toListingSummary(item shared.ListingSummary) *shared.ListingSummary {
	return &shared.ListingSummary{
		Id:           item.Id,
		DealerId:     item.DealerId,
		Make:         item.Make,
		Model:        item.Model,
		Year:         item.Year,
		Price:        item.Price,
		Mileage:      item.Mileage,
		FuelType:     item.FuelType,
		BodyClass:    item.BodyClass,
		DriveType:    item.DriveType,
		Transmission: item.Transmission,
		IsNew:        item.IsNew,
		IsSold:       item.IsSold,
		City:         item.City,
		District:     item.District,
		State:        item.State,
		Country:      item.Country,
		LastSeen:     item.LastSeen,
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	ctx := context.Background()
	shutdownTracing := observability.InitTracing(ctx, logger, "search")
	defer func() {
		if err := shutdownTracing(ctx); err != nil {
			logger.Error("failed to shutdown tracing", "error", err)
		}
	}()

	listingDsn := utils.MustGetEnv("LISTING_DATABASE_URL")

	listingDB := utils.TryConnectDB(listingDsn, 8, 10)

	grpcPort := utils.MustGetEnv("SEARCH_GRPC_PORT")

	lis := utils.TryListen(grpcPort)

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	repo := repository.NewSearchRepository(listingDB)

	redisHost := utils.GetEnv("REDIS_HOST", "redis-cache")
	redisPort := utils.GetEnv("REDIS_PORT", "6379")
	ttlStr := utils.GetEnv("SEARCH_CACHE_TTL_SECONDS", "300")
	ttlSeconds, _ := strconv.Atoi(ttlStr)
	redisCache := cache.NewRedisCache(redisHost, redisPort, time.Duration(ttlSeconds)*time.Second)

	searchService := service.NewSearchService(repo, redisCache)
	proto.RegisterSearchServiceServer(grpcServer, &server{service: searchService})

	utils.HealthCheck("search.SearchService", grpcServer)

	logger.Info("Search gRPC server is running on " + lis.Addr().String() + "...")

	utils.TryServe(grpcServer, lis)
}
