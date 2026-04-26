package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net"
	"os"
	"time"

	marketpricepb "services/marketprice/proto"
	"services/marketprice/service"

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

func connectDB(dsn string, logger *slog.Logger) (*sql.DB, error) {
	var err error
	for i := 0; i < 10; i++ {
		db, openErr := sql.Open("postgres", dsn)
		if openErr == nil {
			if pingErr := db.Ping(); pingErr == nil {
				return db, nil
			} else {
				err = pingErr
			}
		} else {
			err = openErr
		}
		logger.Warn("waiting for database", "attempt", i+1)
		time.Sleep(3 * time.Second)
	}

	return nil, err
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	dsn := os.Getenv("LISTING_DATABASE_URL")
	if dsn == "" {
		logger.Error("LISTING_DATABASE_URL is not set")
		return
	}

	db, err := connectDB(dsn, logger)
	if err != nil {
		logger.Error("failed to connect database", "error", err)
		return
	}

	grpcPort := os.Getenv("MARKETPRICE_GRPC_PORT")
	if grpcPort == "" {
		logger.Error("MARKETPRICE_GRPC_PORT is not set")
		return
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		logger.Error("failed to listen", "error", err)
		return
	}

	grpcServer := grpc.NewServer()
	marketSvc := service.NewService(db)
	marketpricepb.RegisterMarketPriceServiceServer(grpcServer, &server{service: marketSvc, logger: logger})

	logger.Info("Market Price Analysis gRPC server is running", "addr", lis.Addr().String())

	if err := grpcServer.Serve(lis); err != nil {
		logger.Error("failed to serve", "error", err)
	}
}
