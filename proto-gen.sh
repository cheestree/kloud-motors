cd code/services/user
protoc --go_out=. --go-grpc_out=. user.proto
cd ../../../

cd code/services/seller
protoc --go_out=. --go-grpc_out=. seller.proto
cd ../../../