package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	proto "services/auction/proto"
	auctionpubsub "services/auction/pubsub"
	ws2 "services/auction/ws"
	listingproto "services/listing/proto"

	"google.golang.org/grpc/credentials/insecure"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

var db *sql.DB

func initDB() {
	dsn := os.Getenv("AUCTION_DATABASE_URL")
	if dsn == "" {
		log.Fatalf("AUCTION_DATABASE_URL is not set")
	}

	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", dsn)
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				log.Println("Connected to auction database!")
				return
			}
		}
		log.Printf("Waiting for auction database... (%d/10)", i+1)
		time.Sleep(3 * time.Second)
	}

	log.Fatalf("failed to connect database: %v", err)
}

func main() {
	initDB()
	listingAddr := getenv("LISTING_GRPC_ADDR", "listing:50052")
	auctionGRPCPort := getenv("AUCTION_GRPC_PORT", "50056")

	listingConn, err := grpc.NewClient(listingAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to listing service: %v", err)
	}
	defer listingConn.Close()
	listingClient := listingproto.NewListingServiceClient(listingConn)

	nodeID := getenv("POD_ID", localNodeID())
	ps, err := auctionpubsub.NewGCPPubSub(context.Background(), auctionpubsub.GCPPubSubConfig{
		ProjectID:       getenv("GCP_PROJECT_ID", ""),
		TopicID:         getenv("GCP_PUBSUB_TOPIC", "auction-events"),
		SubscriptionID:  getenv("GCP_PUBSUB_SUBSCRIPTION", "auction-sub-"+nodeID),
		NodeID:          nodeID,
		CreateResources: getenvBool("GCP_PUBSUB_AUTOCREATE", false),
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

	lis, err := net.Listen("tcp", ":"+auctionGRPCPort)
	if err != nil {
		log.Fatalf("Error on listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterAuctionServiceServer(grpcServer, &server{
		hub:           hub,
		listingClient: listingClient,
	})

	go func() {
		log.Println("Auction gRPC server is running on " + lis.Addr().String() + "...")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	wsSrv := &wsServer{hub: hub}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws/auction/{auctionID}", wsSrv.ServeWS)
	wsPort := getenv("AUCTION_WS_PORT", "8080")

	log.Printf("Auction WS server is running on :%s...", wsPort)
	if err := http.ListenAndServe(":"+wsPort, mux); err != nil {
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

func localNodeID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "auction-local"
	}
	return host
}
