module github.com/harryrose/godm/downloader

go 1.21.0

require (
	github.com/kelseyhightower/envconfig v1.4.0
	golang.org/x/time v0.3.0
	google.golang.org/grpc v1.58.2
	google.golang.org/protobuf v1.31.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
)

require (
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/harryrose/godm/log v0.0.0
	golang.org/x/net v0.12.0 // indirect
	golang.org/x/sys v0.10.0 // indirect
	golang.org/x/text v0.11.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230711160842-782d3b101e98 // indirect
)

replace github.com/harryrose/godm/log => ../log
