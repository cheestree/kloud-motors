package main

import (
	"context"
	"log/slog"
	"os"

	marketpricepb "services/marketprice/proto"
	"services/marketprice/service"
	"services/utils"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

type server struct {
	marketpricepb.MarketPriceServiceServer
	service *service.Service
	logger  *slog.Logger
}

func (s *server) GetAverageMarketPrice(ctx context.Context, req *marketpricepb.AveragePriceRequest) (*marketpricepb.AveragePriceResponse, error) {
	resp, err := s.service.GetAverageMarketPrice(ctx, req)
	if err != nil {
		s.logger.Error("get average market price failed", "error", err)
		return nil, err
	}
	return resp, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	listingDsn := utils.MustGetEnv("LISTING_DATABASE_URL")

	listingDB := utils.TryConnectDB(listingDsn, 3, 10)

	marketpriceGrpcPort := utils.MustGetEnv("MARKETPRICE_GRPC_PORT")

	lis := utils.TryListen(marketpriceGrpcPort)

	grpcServer := grpc.NewServer()
	marketSvc := service.NewService(listingDB)
	marketpricepb.RegisterMarketPriceServiceServer(grpcServer, &server{service: marketSvc, logger: logger})

	utils.HealthCheck("marketprice.MarketPriceService", grpcServer)

	logger.Info("Market Price Analysis gRPC server is running", "addr", lis.Addr().String())

	utils.TryServe(grpcServer, lis)
}
