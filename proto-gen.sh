cd code/services/user
protoc --go_out=. --go-grpc_out=. user.proto
cd ../../