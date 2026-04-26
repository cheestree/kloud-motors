package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"services/chat/proto"
	pubsub2 "services/chat/pubsub"
	"services/chat/repository"
	"services/chat/repository/firestore"
	"services/chat/repository/postgres"
	ws2 "services/chat/ws"
	listingproto "services/listing/proto"
	sellerproto "services/seller/proto"
	"services/utils"
	"strconv"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type serviceClients struct {
	listingConn   *grpc.ClientConn
	sellerConn    *grpc.ClientConn
	listingClient listingproto.ListingServiceClient
	sellerClient  sellerproto.SellerServiceClient
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()

	nodeID := utils.GetEnv("POD_ID", utils.LocalNodeID())
	pubsub, err := pubsub2.NewGCPPubSub(ctx, pubsub2.GCPPubSubConfig{
		ProjectID:       utils.GetEnv("GCP_PROJECT_ID", ""),
		TopicID:         utils.GetEnv("GCP_PUBSUB_TOPIC", "chat-events"),
		SubscriptionID:  utils.GetEnv("GCP_PUBSUB_SUBSCRIPTION", "chat-sub-"+nodeID),
		NodeID:          nodeID,
		CreateResources: utils.GetEnvBool("GCP_PUBSUB_AUTOCREATE", false),
	})

	if err != nil {
		logger.Error("pubsub init failed, continuing without distributed WS", "error", err)
	}
	if pubsub != nil {
		defer pubsub.Close()
	}

	firestoreProjectID := utils.GetEnv("FIREBASE_PROJECT_ID", utils.GetEnv("GCP_PROJECT_ID", ""))
	var messageRepo repository.MessageRepo
	firestoreRepo, err := firestore.NewFirestoreMessageRepo(
		ctx,
		firestoreProjectID,
		utils.GetEnv("FIRESTORE_MESSAGES_COLLECTION", "messages"),
	)
	if err != nil {
		logger.Error("firestore init failed, continuing without message persistence", "error", err)
	} else {
		messageRepo = firestoreRepo
		defer messageRepo.Close()
	}

	repoConfig := repository.DBConfig{
		Host:   utils.GetEnv("POSTGRES_DSN", ""),
		Schema: utils.GetEnv("POSTGRES_SCHEMA", "chat-db"),
		Table:  utils.GetEnv("POSTGRES_TABLE", "chat"),
	}
	relationalRepo, err := postgres.NewPostgresRepo(ctx, repoConfig)

	if err != nil {
		logger.Error("postgres init failed", "error", err)
	}
	defer relationalRepo.Close()

	hub := ws2.NewHub(pubsub)

	clients, err := setupServiceClients()
	if err != nil {
		logger.Error("service clients init failed", "error", err)
	}
	defer clients.listingConn.Close()
	defer clients.sellerConn.Close()

	if err := setupGRPC(messageRepo, relationalRepo, clients); err != nil {
		logger.Error("grpc setup failed", "error", err)
	}

	if err := setupHTTPWS(hub, messageRepo, relationalRepo, clients.listingClient); err != nil {
		logger.Error("failed to serve HTTPWS", "error", err)
	}
}

func setupServiceClients() (*serviceClients, error) {
	listingConn, err := grpc.NewClient(
		utils.GetEnv("LISTING_SERVICE_ADDR", "localhost:50054"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	sellerConn, err := grpc.NewClient(
		utils.GetEnv("SELLER_SERVICE_ADDR", "localhost:50057"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		_ = listingConn.Close()
		return nil, err
	}

	return &serviceClients{
		listingConn:   listingConn,
		sellerConn:    sellerConn,
		listingClient: listingproto.NewListingServiceClient(listingConn),
		sellerClient:  sellerproto.NewSellerServiceClient(sellerConn),
	}, nil
}

func setupGRPC(messageStore repository.MessageRepo, indexStore repository.ChatIndexRepo, clients *serviceClients) error {
	grpcPort := normalizePort(utils.GetEnv("CHAT_GRPC_PORT", "50052"), "50052")
	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return err
	}

	grpcSrv := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcSrv, &grpcServer{
		messageStore:  messageStore,
		indexStore:    indexStore,
		historyLimit:  utils.GetEnvInt32("CHAT_HISTORY_LIMIT", int32(50)),
		listingClient: clients.listingClient,
		sellerClient:  clients.sellerClient,
	})

	healthcheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcSrv, healthcheck)
	healthcheck.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	go func() {
		log.Printf("gRPC listening on :%s", grpcPort)
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	return nil
}

func setupHTTPWS(hub *ws2.Hub, messageStore repository.MessageRepo, indexStore repository.ChatIndexRepo, listingClient listingproto.ListingServiceClient) error {
	ws := &wsServer{hub: hub, messageStore: messageStore, indexStore: indexStore, listingClient: listingClient}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/chat/{chatID}", ws.ServeWS)

	httpWSPort := normalizePort(utils.GetEnv("CHAT_WS_PORT", "8080"), "8080")

	log.Printf("WS listening on %s", httpWSPort)
	return http.ListenAndServe(":"+httpWSPort, mux)
}

func normalizePort(raw, fallback string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fallback
	}

	if strings.Contains(v, ":") {
		if _, port, err := net.SplitHostPort(v); err == nil && port != "" {
			return port
		}
		idx := strings.LastIndex(v, ":")
		if idx >= 0 && idx+1 < len(v) {
			v = v[idx+1:]
		}
	}

	if _, err := strconv.Atoi(v); err != nil {
		return fallback
	}

	return v
}
