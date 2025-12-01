package main

import (
	"distributed-system/pkg/model"
	"distributed-system/pkg/protocol"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"time"
)

var (
	port  = flag.String("port", "9001", "Port to listen on")
	id    = flag.String("id", "Shard-A", "Shard ID")
	delay = flag.Int("delay", 100, "Artificial processing delay in ms (to demonstrate parallelism)")
)

// BD em memória
var db []model.Transaction

func main() {
	flag.Parse()

	// Popular com dados fictícios
	populateDB()

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		panic(err)
	}
	fmt.Printf("History Shard %s running on :%s with %d records (Delay: %dms)\n", *id, *port, len(db), *delay)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleRequest(conn)
	}
}

func populateDB() {
	// Adicionar transações fictícias
	for i := 0; i < 10; i++ {
		db = append(db, model.Transaction{
			ID:        fmt.Sprintf("%s-%d", *id, i),
			Symbol:    "PETR4",
			Price:     20.0 + float64(i),
			Quantity:  100 * (i + 1),
			Timestamp: time.Now().Add(time.Duration(-i) * time.Hour),
		})
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	var msg protocol.Message
	if err := protocol.ReceiveJSON(conn, &msg); err != nil {
		return
	}

	if msg.Type == protocol.MsgReqHistory {
		// Simular processamento (I/O Bound ou CPU Bound) para evidenciar paralelismo
		if *delay > 0 {
			time.Sleep(time.Duration(*delay) * time.Millisecond)
		}

		// Retornar todos os dados (Simular Consulta)
	
payload, _ := json.Marshal(db)
		resp := protocol.Message{
			Type:    protocol.MsgRespHistory,
			Payload: payload,
		}
		protocol.SendJSON(conn, resp)
		fmt.Printf("[%s] Served history request (latency: %dms)\n", *id, *delay)
	}
}