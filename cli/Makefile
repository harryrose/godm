
.PHONY: client_gen build gen

gen:  client_gen

client_gen:
	protoc --proto_path=../protos/ \
	       --go_out=. \
	       --go_opt=Mqueue.proto=github.com/harryrose/godm/cli/queue \
   	       --go_opt=Mqueue-service.proto=github.com/harryrose/godm/cli/queue \
	       --go_opt=module=github.com/harryrose/godm/cli \
   	       --go-grpc_out=. \
   	       --go-grpc_opt=module=github.com/harryrose/godm/cli \
   	       --go-grpc_opt=Mqueue-service.proto=github.com/harryrose/godm/cli/queue \
   	       --go-grpc_opt=Mqueue.proto=github.com/harryrose/godm/cli/queue \
   	       queue-service.proto \
   	       queue.proto

build:
	go build -o ../build/$(GOOS)/$(GOARCH)/cli cmd/cli/cli.go