#protoc --proto_path=. --go_out=plugins=grpc:. proto/*.proto


protoc --proto_path=. --go_out=. --go-grpc_out=. proto/*.proto
#protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/*.proto


protoc --proto_path=. --grpc-gateway_out=. \
    --grpc-gateway_opt=logtostderr=true \
    --grpc-gateway_opt=generate_unbound_methods=true \
    proto/*.proto

protoc --proto_path=. --openapiv2_out=. \
    --openapiv2_opt=logtostderr=true \
    --openapiv2_opt=generate_unbound_methods=true \
    proto/*.proto


protoc --proto_path=. --go_out=. --go-grpc_out=. \
  --grpc-gateway_out=. --grpc-gateway_opt=logtostderr=true --grpc-gateway_opt=generate_unbound_methods=true \
  --openapiv2_out=. --openapiv2_opt=logtostderr=true --openapiv2_opt=generate_unbound_methods=true --openapiv2_opt=use_go_templates=true \
  proto/*.proto