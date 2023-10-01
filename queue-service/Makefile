
.PHONY: api_gen db_gen build gen

gen: api_gen db_gen

api_gen:
	protoc --proto_path=../protos/ \
	       --go_out=. \
	       --go_opt=Mqueue.proto=github.com/harryrose/godm/queue-service/queue \
   	       --go_opt=Mqueue-service.proto=github.com/harryrose/godm/queue-service/rpc \
	       --go_opt=module=github.com/harryrose/godm/queue-service \
   	       --go-grpc_out=. \
   	       --go-grpc_opt=module=github.com/harryrose/godm/queue-service \
   	       --go-grpc_opt=Mqueue-service.proto=github.com/harryrose/godm/queue-service/rpc \
   	       --go-grpc_opt=Mqueue.proto=github.com/harryrose/godm/queue-service/queue \
   	       queue-service.proto \
   	       queue.proto

db_gen:
	protoc --proto_path=db/protos/ \
	       --go_out=. \
	       --go_opt=Mdb.proto=github.com/harryrose/godm/queue-service/db \
	       --go_opt=module=github.com/harryrose/godm/queue-service \
   	       db.proto

build:
	go build -o ../build/$(GOOS)/$(GOARCH)/queue-service cmd/queue-service/queue-service.go