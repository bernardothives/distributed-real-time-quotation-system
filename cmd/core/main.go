package main

import (
	"distributed-system/pkg/circuitbreaker"
	"distributed-system/pkg/model"
	"distributed-system/pkg/protocol"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	ExternalServiceAddr = "localhost:8080"
	BrokerServiceAddr   = "localhost:8081"
	CoreServicePort     = ":8082"
)

// BrokerClient manages the connection to the Pub/Sub Broker safely.
type BrokerClient struct {
	addr string
	conn net.Conn
	mu   sync.Mutex // Protects connection access (Atomic Writes & Reconnection)
}

func NewBrokerClient(addr string) *BrokerClient {
	return &BrokerClient{
		addr: addr,
	}
}

// Publish sends a message to the broker with automatic reconnection logic.
func (bc *BrokerClient) Publish(topic string, payload []byte) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 1. Ensure Connection
	if bc.conn == nil {
		if err := bc.connect(); err != nil {
			return fmt.Errorf("broker offline: %v", err)
		}
	}

	// 2. Prepare Message inside the lock to ensure sequence
	msg := protocol.Message{
		Type:    protocol.MsgPublish,
		Topic:   topic,
		Payload: payload,
	}

	// 3. Attempt Send
	err := protocol.SendJSON(bc.conn, msg)
	if err != nil {
		fmt.Printf("[Core] Error publishing to broker: %v. Attempting reconnection...\n", err)
		
		// 4. Reconnection Logic (On Error)
		bc.conn.Close()
		bc.conn = nil // Force full reconnect next time or now

		// Try to reconnect immediately once
		if reconErr := bc.connect(); reconErr != nil {
			return fmt.Errorf("reconnection failed: %v", reconErr)
		}

		// Retry Send
		fmt.Println("[Core] Reconnected to Broker. Retrying publish...")
		if err := protocol.SendJSON(bc.conn, msg); err != nil {
			// If fails again, give up for this request to avoid blocking too long
			bc.conn.Close()
			bc.conn = nil
			return fmt.Errorf("retry failed: %v", err)
		}
	}

	return nil
}

func (bc *BrokerClient) connect() error {
	conn, err := net.DialTimeout("tcp", bc.addr, 500*time.Millisecond)
	if err != nil {
		return err
	}
	bc.conn = conn
	return nil
}

func main() {
	// Initialize Circuit Breaker
	cb := circuitbreaker.NewCircuitBreaker(3, 5*time.Second)
	
	// Initialize Robust Broker Client
	brokerClient := NewBrokerClient(BrokerServiceAddr)
	// Attempt initial connection (optional, allows fail-fast check)
	go func() {
		if err := brokerClient.Publish("healthcheck", []byte{}); err != nil {
			fmt.Println("[Core] Initial broker check failed (will retry on demand):", err)
		}
	}()

	// Server for Aggregator
	listener, err := net.Listen("tcp", CoreServicePort)
	if err != nil {
		panic(err)
	}
	fmt.Println("Core Service running on", CoreServicePort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleRequest(conn, cb, brokerClient)
	}
}

func handleRequest(clientConn net.Conn, cb *circuitbreaker.CircuitBreaker, broker *BrokerClient) {
	defer clientConn.Close()

	var msg protocol.Message
	if err := protocol.ReceiveJSON(clientConn, &msg); err != nil {
		return
	}

	if msg.Type == protocol.MsgRequestQuote {
		// Use Circuit Breaker to fetch from External
		result, err := cb.Execute(func() (interface{}, error) {
			return fetchQuoteFromExternal()
		})

		if err != nil {
			errMsg := protocol.NewMessage(protocol.MsgError, err.Error())
			protocol.SendJSON(clientConn, errMsg)
			return
		}

		quote := result.(model.Quote)
		
		// 1. Return to Client (Aggregator)
		respPayload, _ := json.Marshal(quote)
		resp := protocol.Message{Type: protocol.MsgRespQuote, Payload: respPayload}
		protocol.SendJSON(clientConn, resp)

		// 2. Publish to Broker (Robust & Async-ish)
		// We do this synchronously here to ensure order, but since we use a timeout in connect, it won't hang forever.
		err = broker.Publish(quote.Symbol, respPayload)
		if err != nil {
			// Log only, do not fail the client request because pub/sub is auxiliary
			fmt.Println("[Core] Warning: Failed to publish quote:", err)
		} else {
			fmt.Printf("[Core] Published %s to Broker\n", quote.Symbol)
		}
	}
}

func fetchQuoteFromExternal() (model.Quote, error) {
	conn, err := net.DialTimeout("tcp", ExternalServiceAddr, 2*time.Second)
	if err != nil {
		return model.Quote{}, err
	}
	defer conn.Close()

	req := protocol.NewMessage(protocol.MsgRequestQuote, nil)
	if err := protocol.SendJSON(conn, req); err != nil {
		return model.Quote{}, err
	}

	var resp protocol.Message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := protocol.ReceiveJSON(conn, &resp); err != nil {
		return model.Quote{}, err
	}

	var quote model.Quote
	if err := json.Unmarshal(resp.Payload, &quote); err != nil {
		return model.Quote{}, err
	}
	return quote, nil
}