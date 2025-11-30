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

// BrokerClient gerencia a conexão com o Broker Pub/Sub de forma segura.
type BrokerClient struct {
	addr string
	conn net.Conn
	mu   sync.Mutex // Protege o acesso à conexão (Escritas Atômicas & Reconexão)
}

func NewBrokerClient(addr string) *BrokerClient {
	return &BrokerClient{
		addr: addr,
	}
}

// Publish envia uma mensagem para o broker com lógica de reconexão automática.
func (bc *BrokerClient) Publish(topic string, payload []byte) error {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// 1. Garantir Conexão
	if bc.conn == nil {
		if err := bc.connect(); err != nil {
			return fmt.Errorf("broker offline: %v", err)
		}
	}

	// 2. Preparar Mensagem dentro do bloqueio para garantir sequência
	msg := protocol.Message{
		Type:    protocol.MsgPublish,
		Topic:   topic,
		Payload: payload,
	}

	// 3. Tentar Enviar
	err := protocol.SendJSON(bc.conn, msg)
	if err != nil {
		fmt.Printf("[Core] Error publishing to broker: %v. Attempting reconnection...\n", err)
		
		// 4. Lógica de Reconexão (Em caso de Erro)
		bc.conn.Close()
		bc.conn = nil // Forçar reconexão completa na próxima vez ou agora

		// Tentar reconectar imediatamente uma vez
		if reconErr := bc.connect(); reconErr != nil {
			return fmt.Errorf("reconnection failed: %v", reconErr)
		}

		// Tentar Enviar Novamente
		fmt.Println("[Core] Reconnected to Broker. Retrying publish...")
		if err := protocol.SendJSON(bc.conn, msg); err != nil {
			// Se falhar novamente, desistir desta requisição para evitar bloqueio muito longo
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
	// Inicializar Circuit Breaker
	cb := circuitbreaker.NewCircuitBreaker(3, 5*time.Second)
	
	// Inicializar Cliente Broker Robusto
	brokerClient := NewBrokerClient(BrokerServiceAddr)
	// Tentar conexão inicial (opcional, permite verificação rápida de falha)
	go func() {
		if err := brokerClient.Publish("healthcheck", []byte{}); err != nil {
			fmt.Println("[Core] Initial broker check failed (will retry on demand):", err)
		}
	}()

	// Servidor para o Agregador
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
		// Usar Circuit Breaker para buscar do Externo
		result, err := cb.Execute(func() (interface{}, error) {
			return fetchQuoteFromExternal()
		})

		if err != nil {
			errMsg := protocol.NewMessage(protocol.MsgError, err.Error())
			protocol.SendJSON(clientConn, errMsg)
			return
		}

		quote := result.(model.Quote)
		
		// 1. Retornar ao Cliente (Agregador)
		respPayload, _ := json.Marshal(quote)
		resp := protocol.Message{Type: protocol.MsgRespQuote, Payload: respPayload}
		protocol.SendJSON(clientConn, resp)

		// 2. Publicar no Broker (Robusto & Quase Assíncrono)
		// Fazemos isso de forma síncrona aqui para garantir a ordem, mas como usamos um timeout na conexão, não ficará travado para sempre.
		err = broker.Publish(quote.Symbol, respPayload)
		if err != nil {
			// Apenas logar, não falhar a requisição do cliente pois o pub/sub é auxiliar
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