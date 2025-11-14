module github.com/saidutt46/switchboard-gateway

go 1.25

require (
	github.com/lib/pq v1.10.9 // PostgreSQL driver
	github.com/redis/go-redis/v9 v9.5.1 // Redis client
	github.com/segmentio/kafka-go v0.4.47 // Kafka client
)

require (
	github.com/cespare/xxhash/v2 v2.2.0 // indirect (redis dependency)
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect (redis dependency)
	github.com/klauspost/compress v1.17.7 // indirect (kafka dependency)
	github.com/pierrec/lz4/v4 v4.1.21 // indirect (kafka dependency)
)