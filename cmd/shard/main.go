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
	port = flag.String("port", "9001", "Port to listen on")
	id   = flag.String("id", "Shard-A", "Shard ID")
)

// In-memory DB
var db []model.Transaction

func main() {
	flag.Parse()

	// Populate with dummy data
	populateDB()

	listener, err := net.Listen("tcp", ":"+*port)
	if err != nil {
		panic(err)
	}
	fmt.Printf("History Shard %s running on :%s with %d records\n", *id, *port, len(db))

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleRequest(conn)
	}
}

func populateDB() {
	// Add dummy transactions
	// In a real scenario, Shards would own specific keys. 
	// Here we just add data tagged with this Shard ID for demo.
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
		// Return all data (Simulate Query)
		payload, _ := json.Marshal(db)
		resp := protocol.Message{
			Type:    protocol.MsgRespHistory,
			Payload: payload,
		}
		protocol.SendJSON(conn, resp)
		fmt.Printf("[%s] Served history request\n", *id)
	}
}
