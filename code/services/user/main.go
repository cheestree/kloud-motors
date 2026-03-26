package main

import (
    "context"
    proto "user/proto"
    "log"
    "net"

    "google.golang.org/grpc"
)

func main() {
    lis, err := net.Listen("tcp", ":50053")
    if err != nil {
        log.Fatalf("Error on listen: %v", err)
    }

    grpcServer := grpc.NewServer()
    proto.RegisterUserServiceServer(grpcServer, &server{})

    log.Println("User gRPC server is running on " + lis.Addr().String() + "...")

    if err := grpcServer.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
