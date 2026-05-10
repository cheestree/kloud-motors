package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"

	proto "services/auction/proto"
	auctionpubsub "services/auction/pubsub"
	ws2 "services/auction/ws"
	listingproto "services/listing/proto"
	"services/observability"
	utils "services/utils"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

var auctionDB *sql.DB

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	shutdownTracing := observability.InitTracing(ctx, logger, "auction")
	defer func() {
		if err := shutdownTracing(ctx); err != nil {
			logger.Error("failed to shutdown tracing", "error", err)
		}
	}()

	auctionDSN := utils.MustGetEnv("AUCTION_DATABASE_URL")
	auctionDB = utils.TryConnectDB(auctionDSN, 8, 10)

	listingAddr := utils.GetEnv("LISTING_GRPC_ADDR", "listing:50054")
	auctionGRPCPort := utils.GetEnv("AUCTION_GRPC_PORT", "50051")

	listingConn, err := grpc.NewClient(
		listingAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to listing service: %v", err)
	}
	defer listingConn.Close()
	listingClient := listingproto.NewListingServiceClient(listingConn)

	nodeID := utils.GetEnv("POD_ID", utils.LocalNodeID())
	ps, err := auctionpubsub.NewGCPPubSub(ctx, auctionpubsub.GCPPubSubConfig{
		ProjectID:       utils.GetEnv("GCP_PROJECT_ID", ""),
		TopicID:         utils.GetEnv("AUCTION_PUBSUB_TOPIC", "auction-events"),
		SubscriptionID:  utils.GetEnv("AUCTION_PUBSUB_SUBSCRIPTION", "auction-sub-"+nodeID),
		NodeID:          nodeID,
		CreateResources: utils.GetEnvBool("GCP_PUBSUB_AUTOCREATE", false),
	})
	if err != nil {
		log.Printf("pubsub init failed (running without distributed WS): %v", err)
	}

	var hub *ws2.Hub
	if ps != nil {
		defer ps.Close()
		hub = ws2.NewHub(ps)
	} else {
		hub = ws2.NewHub(nil)
	}

	lis := utils.TryListen(auctionGRPCPort)

	grpcServer := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	proto.RegisterAuctionServiceServer(grpcServer, &server{
		hub:           hub,
		listingClient: listingClient,
	})

	utils.HealthCheck("auction.AuctionService", grpcServer)

	go func() {
		log.Println("Auction gRPC server is running on " + lis.Addr().String() + "...")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	wsSrv := &wsServer{hub: hub}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/auction/{auctionID}", wsSrv.ServeWS)
	wsPort := utils.GetEnv("AUCTION_WS_PORT", "8080")

	log.Printf("Auction WS server is running on :%s...", wsPort)
	handler := otelhttp.NewHandler(mux, "auction-websocket")
	if err := http.ListenAndServe(":"+wsPort, handler); err != nil {
		log.Fatalf("http serve: %v", err)
	}
}
