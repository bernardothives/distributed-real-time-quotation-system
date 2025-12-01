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
const (
	coreAddr       = "localhost:8082"
	requestTimeout = 2 * time.Second // Timeout rigoroso para evitar travamentos
)

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
			// Falha parcial aceitável
			errMsg := fmt.Sprintf("Core: %v", err)
			fmt.Println("Error fetching from Core:", errMsg)
			resp.Errors = append(resp.Errors, errMsg)
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
				// Falha parcial aceitável
				errMsg := fmt.Sprintf("Shard(%s): %v", addr, err)
				fmt.Println("Error fetching from Shard:", errMsg)
				resp.Errors = append(resp.Errors, errMsg)
			} else {
				resp.History = append(resp.History, txs...)
			}
			mu.Unlock()
		}(shardAddr)
	}

	// 3. Gather: Aguardar todos
	wg.Wait()
	
	duration := time.Since(start)
	fmt.Printf("Scatter/Gather finished in %v. Errors: %d\n", duration, len(resp.Errors))

	// Enviar de volta ao cliente
	if err := json.NewEncoder(conn).Encode(resp); err != nil {
		fmt.Println("Error sending response to client:", err)
	}
}

func getQuoteFromCore() (model.Quote, error) {
	// Usar DialTimeout para evitar hang na conexão inicial (TCP handshake)
	conn, err := net.DialTimeout("tcp", coreAddr, requestTimeout)
	if err != nil {
		return model.Quote{}, err
	}
	defer conn.Close()

	// Definir Deadline total para a operação (escrita + leitura)
	conn.SetDeadline(time.Now().Add(requestTimeout))

	req := protocol.NewMessage(protocol.MsgRequestQuote, nil)
	if err := protocol.SendJSON(conn, req); err != nil {
		return model.Quote{}, err
	}

	var msg protocol.Message
	if err := protocol.ReceiveJSON(conn, &msg); err != nil {
		return model.Quote{}, err
	}
	
	if msg.Type == protocol.MsgError {
		return model.Quote{}, fmt.Errorf(string(msg.Payload))
	}

	var quote model.Quote
	if err := json.Unmarshal(msg.Payload, &quote); err != nil {
		return model.Quote{}, err
	}
	return quote, nil
}

func getHistoryFromShard(addr string) ([]model.Transaction, error) {
	// Usar DialTimeout
	conn, err := net.DialTimeout("tcp", addr, requestTimeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Definir Deadline
	conn.SetDeadline(time.Now().Add(requestTimeout))

	req := protocol.NewMessage(protocol.MsgReqHistory, nil)
	if err := protocol.SendJSON(conn, req); err != nil {
		return nil, err
	}

	var msg protocol.Message
	if err := protocol.ReceiveJSON(conn, &msg); err != nil {
		return nil, err
	}

	var txs []model.Transaction
	if err := json.Unmarshal(msg.Payload, &txs); err != nil {
		return nil, err
	}
	return txs, nil
}