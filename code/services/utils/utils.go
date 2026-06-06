package utils

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GetEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func MustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return v
}

func GetEnvInt(key string, fallback int) int {
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

func GetEnvBool(key string, fallback bool) bool {
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

func LocalNodeID() string {
	host, err := os.Hostname()
	if err != nil || host == "" {
		return "auction-local"
	}
	return host
}

func GetEnvInt32(key string, fallback int32) int32 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return int32(parsed)
}

func StringPtrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func PositiveInt32Ptr(value int32) *int32 {
	if value <= 0 {
		return nil
	}
	return &value
}

func PositiveInt64PtrFromInt32(value int32) *int64 {
	if value <= 0 {
		return nil
	}
	converted := int64(value)
	return &converted
}

func PositiveInt64PtrFromFloat(value float64) *int64 {
	if value <= 0 {
		return nil
	}
	converted := int64(value)
	return &converted
}

func SQLNullableNonEmptyString(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func SQLNullablePositiveInt32(value int32) interface{} {
	if value <= 0 {
		return nil
	}
	return value
}

func SQLNullablePositiveInt64(value int64) interface{} {
	if value <= 0 {
		return nil
	}
	return value
}

func SQLNullableInt64FromPtr(value *int64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func Int32ValueFromPtrOrZero(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func Int64ValueFromPtrOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func BoolPtrToProtoBoolValue(value *bool) *wrapperspb.BoolValue {
	if value == nil {
		return nil
	}
	return wrapperspb.Bool(*value)
}

func ParseOptionalBoolProtoBoolValue(raw string) (*wrapperspb.BoolValue, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, err
	}
	return wrapperspb.Bool(value), nil
}

func ParseInt32OrZero(s string) int32 {
	var v int32
	_, _ = fmt.Sscan(s, &v)
	return v
}

func ParseInt32OrDefaultIfEmpty(s string, def int32) int32 {
	if s == "" {
		return def
	}
	return ParseInt32OrZero(s)
}

func ParseInt64OrZero(s string) int64 {
	var v int64
	_, _ = fmt.Sscan(s, &v)
	return v
}

func TryListen(port string) net.Listener {
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}
	return lis
}

func TryServe(grpcServer *grpc.Server, lis net.Listener) {
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func TryConnectDB(databaseURL string, timeout int, tries int) *sql.DB {
	var db *sql.DB
	var err error

	for i := 0; i < tries; i++ {
		db, err = sql.Open("postgres", databaseURL)
		if err == nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = db.PingContext(ctx)
			if err == nil {
				db.SetMaxOpenConns(10)
				db.SetMaxIdleConns(5)
				db.SetConnMaxLifetime(5 * time.Minute)
				db.SetConnMaxIdleTime(2 * time.Minute)

				return db
			}

			db.Close()
		}

		logRetry(i, tries, err, timeout)
	}

	log.Fatalf("Failed to connect to database after %d attempts: %v", tries, err)
	return nil
}

func TryConnectGorm(databaseURL string, timeout int, tries int) *gorm.DB {
	var err error

	for i := 0; i < tries; i++ {
		db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
		if err != nil {
			logRetry(i, tries, err, timeout)
			continue
		}

		sqlDB, err := db.DB()
		if err != nil {
			logRetry(i, tries, err, timeout)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = sqlDB.PingContext(ctx)
		cancel()

		if err == nil {
			sqlDB.SetMaxOpenConns(10)
			sqlDB.SetMaxIdleConns(5)
			sqlDB.SetConnMaxLifetime(5 * time.Minute)
			sqlDB.SetConnMaxIdleTime(2 * time.Minute)

			return db
		}

		sqlDB.Close()
		logRetry(i, tries, err, timeout)
	}

	log.Fatalf("Failed to connect to database after %d attempts: %v", tries, err)
	return nil
}

func logRetry(i, tries int, err error, timeout int) {
	if i < tries-1 {
		log.Printf(
			"Database not ready (attempt %d/%d): %v. Retrying in %ds",
			i+1, tries, err, timeout,
		)
		time.Sleep(time.Duration(timeout) * time.Second)
	}
}

func HealthCheck(service string, grpcServer *grpc.Server) {
	healthcheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthcheck)
	healthcheck.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_SERVING)
}
