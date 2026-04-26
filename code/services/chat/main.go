package main

import (
	"context"
	"log"
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
	ctx := context.Background()

	nodeID := getenv("POD_ID", localNodeID())
	pubsub, err := pubsub2.NewGCPPubSub(ctx, pubsub2.GCPPubSubConfig{
		ProjectID:       getenv("GCP_PROJECT_ID", ""),
		TopicID:         getenv("GCP_PUBSUB_TOPIC", "chat-events"),
		SubscriptionID:  getenv("GCP_PUBSUB_SUBSCRIPTION", "chat-sub-"+nodeID),
		NodeID:          nodeID,
		CreateResources: getenvBool("GCP_PUBSUB_AUTOCREATE", false),
	})

	if err != nil {
		log.Printf("pubsub init failed, continuing without distributed WS: %v", err)
	}
	if pubsub != nil {
		defer pubsub.Close()
	}

	firestoreProjectID := getenv("FIREBASE_PROJECT_ID", getenv("GCP_PROJECT_ID", ""))
	var messageRepo repository.MessageRepo
	firestoreRepo, err := firestore.NewFirestoreMessageRepo(
		ctx,
		firestoreProjectID,
		getenv("FIRESTORE_MESSAGES_COLLECTION", "messages"),
	)
	if err != nil {
		log.Printf("firestore init failed, continuing without message persistence: %v", err)
	} else {
		messageRepo = firestoreRepo
		defer messageRepo.Close()
	}

	repoConfig := repository.DBConfig{
		Host:   getenv("POSTGRES_DSN", ""),
		Schema: getenv("POSTGRES_SCHEMA", "chat-db"),
		Table:  getenv("POSTGRES_TABLE", "chat"),
	}
	relationalRepo, err := postgres.NewPostgresRepo(ctx, repoConfig)

	if err != nil {
		log.Fatalf("postgres init: %v", err)
	}
	defer relationalRepo.Close()

	hub := ws2.NewHub(pubsub)


	clients, err := setupServiceClients()
	if err != nil {
		log.Fatalf("service clients init: %v", err)
	}
	defer clients.listingConn.Close()
	defer clients.sellerConn.Close()

	if err := setupGRPC(messageRepo, relationalRepo, clients); err != nil {
		log.Fatalf("grpc setup: %v", err)
	}

	if err := setupHTTPWS(hub, messageRepo, relationalRepo, clients.listingClient); err != nil {
		log.Fatalf("http serve: %v", err)
	}
}

func setupServiceClients() (*serviceClients, error) {
	listingConn, err := grpc.NewClient(
		getenv("LISTING_SERVICE_ADDR", "localhost:50054"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	sellerConn, err := grpc.NewClient(
		getenv("SELLER_SERVICE_ADDR", "localhost:50057"),
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
	grpcPort := normalizePort(getenv("CHAT_GRPC_PORT", "50052"), "50052")
	grpcLis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return err
	}

	grpcSrv := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcSrv, &grpcServer{
		messageStore:  messageStore,
		indexStore:    indexStore,
		historyLimit:  getenvInt32("CHAT_HISTORY_LIMIT", int32(50)),
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

	httpWSPort := normalizePort(getenv("CHAT_WS_PORT", "8080"), "8080")

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

func getenv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getenvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}

	return b
}

func getenvInt32(key string, fallback int32) int32 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return fallback
	}

	return int32(n)
}

func localNodeID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "chat-local"
	}
	return host
}
