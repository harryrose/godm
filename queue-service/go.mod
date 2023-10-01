module github.com/harryrose/godm/queue-service

go 1.21.0

require (
	github.com/golang/protobuf v1.5.3
	go.etcd.io/bbolt v1.3.7
	google.golang.org/grpc v1.58.2
	google.golang.org/protobuf v1.31.0
)

require github.com/kelseyhightower/envconfig v1.4.0

require (
	github.com/harryrose/godm/log v0.0.0
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
)

replace github.com/harryrose/godm/log => ../log
