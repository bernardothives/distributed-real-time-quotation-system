package main

import (
	"distributed-system/pkg/circuitbreaker"
	"distributed-system/pkg/model"
	"distributed-system/pkg/protocol"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

const (
	ExternalServiceAddr = "localhost:8080"
	BrokerServiceAddr   = "localhost:8081"
	CoreServicePort     = ":8082"
)

func main() {
	cb := circuitbreaker.NewCircuitBreaker(3, 5*time.Second)
	
	// Connect to Broker (Publisher)
	brokerConn, err := net.Dial("tcp", BrokerServiceAddr)
	if err != nil {
		fmt.Println("Warning: Could not connect to Broker:", err)
	} else {
		defer brokerConn.Close()
	}

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
		go handleRequest(conn, cb, brokerConn)
	}
}

func handleRequest(clientConn net.Conn, cb *circuitbreaker.CircuitBreaker, brokerConn net.Conn) {
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
			// Return error or cached fallback
			errMsg := protocol.NewMessage(protocol.MsgError, err.Error())
			protocol.SendJSON(clientConn, errMsg)
			return
		}

		quote := result.(model.Quote)
		
		// 1. Return to Client (Aggregator)
		respPayload, _ := json.Marshal(quote)
		resp := protocol.Message{Type: protocol.MsgRespQuote, Payload: respPayload}
		protocol.SendJSON(clientConn, resp)

		// 2. Publish to Broker (Fire and Forget)
		if brokerConn != nil {
			pubMsg := protocol.Message{
				Type:    protocol.MsgPublish,
				Topic:   quote.Symbol,
				Payload: respPayload,
			}
			// Need to wrap in another envelope because protocol expects Type/Topic
			protocol.SendJSON(brokerConn, pubMsg)
			fmt.Println("Published quote to Broker:", quote.Symbol)
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
	conn.SetReadDeadline(time.Now().Add(2 * time.Second)) // Timeout for read
	if err := protocol.ReceiveJSON(conn, &resp); err != nil {
		return model.Quote{}, err
	}

	var quote model.Quote
	if err := json.Unmarshal(resp.Payload, &quote); err != nil {
		return model.Quote{}, err
	}
	return quote, nil
}
