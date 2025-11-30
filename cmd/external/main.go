package main

import (
	"distributed-system/pkg/model"
	"distributed-system/pkg/protocol"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"
)

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("External Quote Service (Mock) running on :8080")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Connection error:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		var msg protocol.Message
		if err := protocol.ReceiveJSON(conn, &msg); err != nil {
			return
		}

		if msg.Type == protocol.MsgRequestQuote {
			// Simular Caos (Falha ou Atraso)
			chaos := r.Float64()
			if chaos < 0.2 { // 20% de chance de timeout/erro
				fmt.Println("Simulating failure...")
				// Simplesmente fechar a conexão ou enviar lixo simula problemas de rede
				return 
			} else if chaos < 0.4 { // 20% de atraso
				time.Sleep(2 * time.Second)
			}

			// Resposta de Sucesso
			quote := model.Quote{
				Symbol:    "PETR4",
				Price:     20.0 + r.Float64()*10,
				Timestamp: time.Now(),
			}
			
			payload, _ := json.Marshal(quote)
			response := protocol.Message{
				Type:    protocol.MsgRespQuote,
				Payload: payload,
			}
			protocol.SendJSON(conn, response)
		}
	}
}
