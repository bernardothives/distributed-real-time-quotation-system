package main

import (
	"distributed-system/pkg/protocol"
	"encoding/json"
	"flag"
	"fmt"
	"net"
)

func main() {
	mode := flag.String("mode", "aggregator", "Mode: 'aggregator' or 'subscribe'")
	flag.Parse()

	if *mode == "subscribe" {
		runSubscriber()
	} else {
		runAggregatorClient()
	}
}

func runAggregatorClient() {
	conn, err := net.Dial("tcp", "localhost:8000")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("Requesting Aggregated Data...")
	// Just connecting triggers the handler in Aggregator (simple trigger)
	// Or we can send a dummy byte if read blocks.
	// Aggregator expects connection, then handles immediately. Wait, Aggregator reads?
	// In my impl: Aggregator accepts, then handleClient runs. handleClient sends response.
	// It does NOT wait for input. So just connecting is enough.

	decoder := json.NewDecoder(conn)
	var resp map[string]interface{} // Generic map to print pretty
	if err := decoder.Decode(&resp); err != nil {
		panic(err)
	}

	formatted, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(formatted))
}

func runSubscriber() {
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Subscribe
	subMsg := protocol.NewMessage(protocol.MsgSubscribe, nil)
	subMsg.Topic = "PETR4"
	protocol.SendJSON(conn, subMsg)
	fmt.Println("Subscribed to PETR4. Waiting for updates...")

	for {
		var msg protocol.Message
		if err := protocol.ReceiveJSON(conn, &msg); err != nil {
			fmt.Println("Connection closed")
			return
		}
		if msg.Type == protocol.MsgPublish {
			fmt.Printf("Received Update: %s\n", string(msg.Payload))
		}
	}
}
