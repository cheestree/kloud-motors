package utils

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
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

func StringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func Int32Ptr(value int32) *int32 {
	if value <= 0 {
		return nil
	}
	return &value
}

func Int64PtrFromInt32(value int32) *int64 {
	if value <= 0 {
		return nil
	}
	converted := int64(value)
	return &converted
}

func Int64PtrFromFloat(value float64) *int64 {
	if value <= 0 {
		return nil
	}
	converted := int64(value)
	return &converted
}

func NullableString(value string) interface{} {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func NullableInt32(value int32) interface{} {
	if value <= 0 {
		return nil
	}
	return value
}

func NullableInt64(value int64) interface{} {
	if value <= 0 {
		return nil
	}
	return value
}

func NullableInt64Ptr(value *int64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

func ParseInt32(s string) int32 {
	var v int32
	_, _ = fmt.Sscan(s, &v)
	return v
}

func ParseInt32WithDefault(s string, def int32) int32 {
	if s == "" {
		return def
	}
	return ParseInt32(s)
}

func ParseInt64(s string) int64 {
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
	db, err := sql.Open("postgres", databaseURL)

	for i := range tries {
		if err == nil {
			if pingErr := db.Ping(); pingErr == nil {
				return db
			} else {
				err = pingErr
			}
		}
		log.Printf("Waiting for database to be ready (attempt %d/%d)", i+1, tries)
		time.Sleep(time.Duration(timeout) * time.Second)
	}

	log.Fatalf("Failed to connect to database after %d attempts: %v", tries, err)
	return nil
}

func TryConnectGorm(databaseURL string, timeout int, tries int) *gorm.DB {
	var db *gorm.DB
	var err error

	for i := range tries {
		db, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
		if err == nil {
			sqlDB, pingErr := db.DB()
			if pingErr == nil {
				if pingErr = sqlDB.Ping(); pingErr == nil {
					return db
				} else {
					err = pingErr
				}
			} else {
				err = pingErr
			}
		}
		log.Printf("Waiting for database to be ready (attempt %d/%d)", i+1, tries)
		time.Sleep(time.Duration(timeout) * time.Second)
	}

	log.Fatalf("Failed to connect to database after %d attempts: %v", tries, err)
	return nil
}
