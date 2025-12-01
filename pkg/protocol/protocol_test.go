package protocol

import (
	"encoding/json"
	"net"
	"testing"
	"time"
)

// TestMessageSerialization verifica se conseguimos codificar e decodificar mensagens corretamente
func TestMessageSerialization(t *testing.T) {
	payload := []byte(`{"price": 100.50}`)
	originalMsg := NewMessage(MsgPublish, payload)

	// Simular envio e recebimento via Pipe (memória)
	r, w := net.Pipe()

	go func() {
		defer w.Close()
		if err := SendJSON(w, originalMsg); err != nil {
			t.Errorf("Failed to send JSON: %v", err)
		}
	}()

	var receivedMsg Message
	// Timeout de segurança
	r.SetReadDeadline(time.Now().Add(1 * time.Second))
	if err := ReceiveJSON(r, &receivedMsg); err != nil {
		t.Fatalf("Failed to receive JSON: %v", err)
	}

	if receivedMsg.Type != originalMsg.Type {
		t.Errorf("Expected type %s, got %s", originalMsg.Type, receivedMsg.Type)
	}
	if string(receivedMsg.Payload) != string(originalMsg.Payload) {
		t.Errorf("Expected payload %s, got %s", originalMsg.Payload, receivedMsg.Payload)
	}
}

// TestMalformedJSON verifica como o protocolo lida com dados corrompidos
func TestMalformedJSON(t *testing.T) {
	r, w := net.Pipe()

	go func() {
		defer w.Close()
		// Envia lixo bytes que não são JSON
		w.Write([]byte("THIS IS NOT JSON\n"))
	}()

	var msg Message
	r.SetReadDeadline(time.Now().Add(1 * time.Second))
	err := ReceiveJSON(r, &msg)

	// Esperamos um erro aqui, pois JSON decoder deve falhar
	if err == nil {
		t.Error("Expected error for malformed JSON, got nil")
	} else if _, ok := err.(*json.SyntaxError); !ok {
		// O ReceiveJSON usa json.Decoder, então erros de sintaxe são esperados
		// Mas como ReceiveJSON não retorna o erro bruto do decoder diretamente sem wrapper as vezes, 
		// apenas verificar se é erro já é suficiente para este teste básico.
		t.Logf("Received expected error: %v", err)
	}
}
