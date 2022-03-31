## GRPC tryout

### Document
* [Protobuf](https://developers.google.com/protocol-buffers)
* [GRPC](https://www.grpc.io)
* [GRPC Gateway](https://github.com/grpc-ecosystem/grpc-gateway)
* [GRPC Gateway](https://grpc-ecosystem.github.io/grpc-gateway/)
* [a_bit_of_everything.proto](https://github.com/grpc-ecosystem/grpc-gateway/blob/master/examples/internal/proto/examplepb/a_bit_of_everything.proto)

### Tech
* Go language
* Protobuf
* GRPC
* TLS
* HTTP gateway [ref1](https://github.com/grpc-ecosystem/grpc-gateway), [ref2](https://grpc-ecosystem.github.io/grpc-gateway/)
* OpenAPI, swagger
* OAuth, token authentication

#### Install

protoc 產生各種檔案，包含：
* *.pb.go
* *_grpc.pb.go
* *.pb.gw.go
* *.swagger.json

```shell
protoc --proto_path=. --go_out=. --go-grpc_out=. \
  --grpc-gateway_out=. --grpc-gateway_opt=logtostderr=true --grpc-gateway_opt=generate_unbound_methods=true \
  --openapiv2_out=. --openapiv2_opt=logtostderr=true --openapiv2_opt=generate_unbound_methods=true --openapiv2_opt=use_go_templates=true \
  proto/*.proto
```

注意事項
* google, protoc-gen-openapiv2 目錄的原始碼來自於官方 github，能改成自動下載的話會比較好。
* protoc 產生出來的 *.swagger.json 要手動複製 my 目錄裡面，只是程式設計的邏輯存取目錄差異。但為何產生時候不是自動放在 my 目錄。

#### Test
##### Open API
[Swagger Json](http://localhost:8081/openapiv2/MyService.swagger.json)

##### GRPC http gateway (REST API)
```shell
$ curl -X POST -H "Authorization: Bearer client-x-id" -k http://localhost:8081/v1/echo -d '{"message": " hello"}'
{"message":" hello","timestamp":"1648689560"}
```



