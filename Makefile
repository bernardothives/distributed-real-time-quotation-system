all: build

build:
	go build -o bin/external cmd/external/main.go
	go build -o bin/broker cmd/broker/main.go
	go build -o bin/core cmd/core/main.go
	go build -o bin/shard cmd/shard/main.go
	go build -o bin/aggregator cmd/aggregator/main.go
	go build -o bin/client cmd/client/main.go

run-all: build
	@echo "Starting Infrastructure..."
	@./bin/external & echo $! > external.pid
	@./bin/broker & echo $! > broker.pid
	@sleep 1
	@./bin/core & echo $! > core.pid
	@./bin/shard -port=9001 -id=Shard-A & echo $! > shard1.pid
	@./bin/shard -port=9002 -id=Shard-B & echo $! > shard2.pid
	@./bin/shard -port=9003 -id=Shard-C & echo $! > shard3.pid
	@./bin/aggregator & echo $! > aggregator.pid
	@echo "All services started."

stop-all:
	@echo "Stopping all services (Kill by Name)..."
	@-pkill -f bin/external || true
	@-pkill -f bin/broker || true
	@-pkill -f bin/core || true
	@-pkill -f bin/shard || true
	@-pkill -f bin/aggregator || true
	@rm *.pid 2>/dev/null || true
	@echo "All services stopped."

test-aggregator:
	@./bin/client -mode=aggregator

test-sub:
	@./bin/client -mode=subscribe
