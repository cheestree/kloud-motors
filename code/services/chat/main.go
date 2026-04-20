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

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	nodeID := getenv("POD_ID", localNodeID())
	pubsub, err := pubsub2.NewGCPPubSub(context.Background(), pubsub2.GCPPubSubConfig{
		ProjectID:       getenv("GCP_PROJECT_ID", ""),
		TopicID:         getenv("GCP_PUBSUB_TOPIC", "chat-events"),
		SubscriptionID:  getenv("GCP_PUBSUB_SUBSCRIPTION", "chat-sub-"+nodeID),
		NodeID:          nodeID,
		CreateResources: getenvBool("GCP_PUBSUB_AUTOCREATE", false),
	})

	if err != nil {
		log.Fatalf("pubsub init: %v", err)
	}
	defer pubsub.Close()

	firestoreProjectID := getenv("FIREBASE_PROJECT_ID", getenv("GCP_PROJECT_ID", ""))
	messageRepo, err := firestore.NewFirestoreMessageRepo(
		context.Background(),
		firestoreProjectID,
		getenv("FIRESTORE_MESSAGES_COLLECTION", "messages"),
	)
	if err != nil {
		log.Fatalf("firestore init: %v", err)
	}
	defer messageRepo.Close()

	repoConfig := repository.DBConfig{
		Host:   getenv("POSTGRES_DSN", ""),
		Schema: getenv("POSTGRES_SCHEMA", "chat-db"),
		Table:  getenv("POSTGRES_TABLE", "chat"),
	}
	relationalRepo, err := postgres.NewPostgresRepo(context.Background(), repoConfig)

	if err != nil {
		log.Fatalf("postgres init: %v", err)
	}
	defer relationalRepo.Close()

	hub := ws2.NewHub(pubsub)

	// ── gRPC (OpenChat, GetChatHistory) ──────────────────────────────────────
	grpcLis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	listingConn, err := grpc.NewClient(
		getenv("LISTING_SERVICE_ADDR", "localhost:50054"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to listing service: %v", err)
	}
	defer listingConn.Close()

	sellerConn, err := grpc.NewClient(
		getenv("SELLER_SERVICE_ADDR", "localhost:50057"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to seller service: %v", err)
	}
	defer sellerConn.Close()

	listingClient := listingproto.NewListingServiceClient(listingConn)
	sellerClient := sellerproto.NewSellerServiceClient(sellerConn)

	grpcSrv := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcSrv, &grpcServer{
		messageStore:  messageRepo,
		indexStore:    relationalRepo,
		historyLimit:  getenvInt32("CHAT_HISTORY_LIMIT", int32(50)),
		listingClient: listingClient,
		sellerClient:  sellerClient,
	})

	go func() {
		log.Println("gRPC listening on :50052")
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// ── HTTP / WebSocket ──────────────────────────────────────────────────────
	ws := &wsServer{hub: hub, messageStore: messageRepo, indexStore: relationalRepo, listingClient: listingClient}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/chat/{chatID}", ws.ServeWS)

	log.Println("WS listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("http serve: %v", err)
	}
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
