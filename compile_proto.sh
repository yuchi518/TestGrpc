#protoc --proto_path=. --go_out=plugins=grpc:. proto/*.proto


protoc --proto_path=. --go_out=. --go-grpc_out=. proto/*.proto
#protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/*.proto