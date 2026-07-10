module github.com/marginpilot/gateway

go 1.23

require (
	github.com/marginpilot/contracts v0.0.0
	github.com/marginpilot/shared v0.0.0
	github.com/redis/go-redis/v9 v9.7.0
	github.com/segmentio/kafka-go v0.4.47
	google.golang.org/grpc v1.71.0
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250115164207-1a7da9e5054f // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/marginpilot/shared => ../../shared

replace github.com/marginpilot/contracts => ../../proto
