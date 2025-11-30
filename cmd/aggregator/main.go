package main

import (
	"distributed-system/pkg/model"
	"distributed-system/pkg/protocol"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// Configuração
var shards = []string{"localhost:9001", "localhost:9002", "localhost:9003"}
const coreAddr = "localhost:8082"

type AggregatedResponse struct {
	CurrentPrice model.Quote         `json:"current_price"`
	History      []model.Transaction `json:"history"`
	Errors       []string            `json:"errors,omitempty"`
}

func main() {
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		panic(err)
	}
	fmt.Println("Aggregator Service running on :8000")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Received client request, starting Scatter/Gather...")

	start := time.Now()
	
	var resp AggregatedResponse
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 1. Scatter: Serviço Core (Preço Atual)
	wg.Add(1)
	go func() {
		defer wg.Done()
		quote, err := getQuoteFromCore()
		mu.Lock()
		if err != nil {
			resp.Errors = append(resp.Errors, "Core: "+err.Error())
		} else {
			resp.CurrentPrice = quote
		}
		mu.Unlock()
	}()

	// 2. Scatter: Shards de Histórico
	for _, shardAddr := range shards {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			txs, err := getHistoryFromShard(addr)
			mu.Lock()
			if err != nil {
				resp.Errors = append(resp.Errors, "Shard("+addr+"): "+err.Error())
			} else {
				resp.History = append(resp.History, txs...)
			}
			mu.Unlock()
		}(shardAddr)
	}

	// 3. Gather: Aguardar todos
	wg.Wait()
	
	fmt.Printf("Scatter/Gather finished in %v\n", time.Since(start))

	// Enviar de volta ao cliente
	json.NewEncoder(conn).Encode(resp)
}

func getQuoteFromCore() (model.Quote, error) {
	conn, err := net.Dial("tcp", coreAddr)
	if err != nil {
		return model.Quote{}, err
	}
	defer conn.Close()

	req := protocol.NewMessage(protocol.MsgRequestQuote, nil)
	protocol.SendJSON(conn, req)

	var msg protocol.Message
	if err := protocol.ReceiveJSON(conn, &msg); err != nil {
		return model.Quote{}, err
	}
	
	if msg.Type == protocol.MsgError {
		return model.Quote{}, fmt.Errorf(string(msg.Payload))
	}

	var quote model.Quote
	json.Unmarshal(msg.Payload, &quote)
	return quote, nil
}

func getHistoryFromShard(addr string) ([]model.Transaction, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	req := protocol.NewMessage(protocol.MsgReqHistory, nil)
	protocol.SendJSON(conn, req)

	var msg protocol.Message
	if err := protocol.ReceiveJSON(conn, &msg); err != nil {
		return nil, err
	}

	var txs []model.Transaction
	json.Unmarshal(msg.Payload, &txs)
	return txs, nil
}
