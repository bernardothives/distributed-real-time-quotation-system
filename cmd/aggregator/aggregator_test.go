package main

import (
	"distributed-system/pkg/model"
	"distributed-system/pkg/protocol"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// TestGetHistoryFromShard_Resilience valida se o cliente lida com erros de rede e dados.
func TestGetHistoryFromShard_Resilience(t *testing.T) {
	// Setup: Servidor Mock de Shard na porta aleatória
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	mockAddr := listener.Addr().String()

	// Goroutine do Servidor Mock
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return // Teste provavelmente acabou
		}
		defer conn.Close()
		
		// Ler request (ignorar conteúdo para o teste)
		var req protocol.Message
		protocol.ReceiveJSON(conn, &req)

		// Responder com sucesso simulado
		txs := []model.Transaction{{ID: "tx1", Price: 100.50, Symbol: "TEST"}}
		payload, _ := json.Marshal(txs)
		resp := protocol.Message{Type: protocol.MsgReqHistory, Payload: payload}
		
		// Simular latência leve
		time.Sleep(10 * time.Millisecond)
		protocol.SendJSON(conn, resp)
	}()

	// Executar a função alvo (do main.go)
	// Nota: getHistoryFromShard faz dial tcp.
	txs, err := getHistoryFromShard(mockAddr)

	if err != nil {
		t.Fatalf("Falha ao buscar histórico do mock: %v", err)
	}

	if len(txs) != 1 || txs[0].ID != "tx1" {
		t.Errorf("Dados recebidos incorretos. Recebido: %v", txs)
	}
}

// TestGetHistoryFromShard_ConnectionRefused valida comportamento quando o shard está down
func TestGetHistoryFromShard_ConnectionRefused(t *testing.T) {
	// Tentar conectar em uma porta onde esperamos que nada esteja rodando
	// Usando porta alta aleatória e localhost
	_, err := getHistoryFromShard("localhost:45821")
	
	if err == nil {
		t.Error("Esperava erro de conexão recusada, recebeu nil")
	}
}
