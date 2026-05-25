package integrationtest

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func DialGRPC(ctx context.Context, t testing.TB, serviceName, addr string) *grpc.ClientConn {
	t.Helper()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		t.Fatalf("failed to dial %s service at %s: %v", serviceName, addr, err)
	}

	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Fatalf("failed to close %s grpc connection: %v", serviceName, err)
		}
	})

	return conn
}
