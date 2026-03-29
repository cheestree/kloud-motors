package main

import (
	"chat/proto"
	pubsub2 "chat/pubsub"
	"chat/repository/firestore"
	"chat/repository/postgres"
	ws2 "chat/ws"
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	"google.golang.org/grpc"
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
	messageStore, err := firestore.NewFirestoreMessageRepo(
		context.Background(),
		firestoreProjectID,
		getenv("FIRESTORE_MESSAGES_COLLECTION", "messages"),
	)
	if err != nil {
		log.Fatalf("firestore init: %v", err)
	}
	defer messageStore.Close()

	indexStore, err := postgres.NewPostgresIndexRepo(context.Background(), getenv("POSTGRES_DSN", ""))
	if err != nil {
		log.Fatalf("postgres init: %v", err)
	}
	defer indexStore.Close()

	hub := ws2.NewHub(pubsub)

	// ── gRPC (OpenChat, GetChatHistory) ──────────────────────────────────────
	grpcLis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("grpc listen: %v", err)
	}

	grpcSrv := grpc.NewServer()
	proto.RegisterChatServiceServer(grpcSrv, &grpcServer{
		messageStore: messageStore,
		indexStore:   indexStore,
		historyLimit: getenvInt("CHAT_HISTORY_LIMIT", 50),
	})

	go func() {
		log.Println("gRPC listening on :50051")
		if err := grpcSrv.Serve(grpcLis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// ── HTTP / WebSocket ──────────────────────────────────────────────────────
	ws := &wsServer{hub: hub, messageStore: messageStore, indexStore: indexStore}
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

func localNodeID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "chat-local"
	}
	return host
}
