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
	// Apenas conectar aciona o handler no Agregador (gatilho simples)
	// Ou podemos enviar um byte fictício se a leitura bloquear.
	// O Agregador espera a conexão, depois trata imediatamente. Espera, o Agregador lê?
	// Na minha impl: Agregador aceita, então handleClient roda. handleClient envia resposta.
	// Ele NÃO espera entrada. Então apenas conectar é suficiente.

	decoder := json.NewDecoder(conn)
	var resp map[string]interface{} // Mapa genérico para imprimir bonito
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

	// Inscrever-se
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
