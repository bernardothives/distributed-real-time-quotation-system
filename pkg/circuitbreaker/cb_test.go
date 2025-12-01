package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

// TestStateTransitions valida o ciclo completo: Closed -> Open -> HalfOpen -> Closed
func TestStateTransitions(t *testing.T) {
	// Config: 2 falhas para abrir, timeout de reset muito curto (50ms)
	threshold := 2
	resetTimeout := 50 * time.Millisecond
	cb := NewCircuitBreaker(threshold, resetTimeout)

	// Ação simulada que falha
	failAction := func() (interface{}, error) {
		return nil, errors.New("service error")
	}

	// Ação simulada que funciona
	successAction := func() (interface{}, error) {
		return "success", nil
	}

	// 1. Estado Inicial: Closed
	if cb.state != StateClosed {
		t.Fatalf("Estado inicial deve ser Closed, é %v", cb.state)
	}

	// 2. Causar falhas até o limiar
	for i := 0; i < threshold; i++ {
		_, err := cb.Execute(failAction)
		if err == nil {
			t.Error("Esperava erro da ação simulada")
		}
	}

	// 3. Verificar se abriu
	// O Circuit Breaker muda para Open DEPOIS da falha que atinge o threshold.
	// A próxima chamada deve ser rejeitada pelo CB.
	_, err := cb.Execute(successAction)
	if err == nil || err.Error() != "circuit breaker is OPEN" {
		t.Errorf("Esperado erro 'circuit breaker is OPEN', recebeu: %v", err)
	}

	if cb.state != StateOpen {
		t.Errorf("Estado deveria ser Open, é %v", cb.state)
	}

	// 4. Aguardar o timeout de reset para transição Half-Open
	time.Sleep(resetTimeout * 2)

	// A transição para Half-Open acontece *durante* a chamada do Execute quando o tempo passou
	// Então a próxima chamada DEVE passar (como teste/probe)
	
	// 5. Tentar novamente (Probe success)
	result, err := cb.Execute(successAction)
	if err != nil {
		t.Errorf("Esperado sucesso após timeout (Half-Open Probe), mas recebeu erro: %v", err)
	}
	if result != "success" {
		t.Errorf("Resultado incorreto recebido")
	}

	// 6. Verificar se o circuito fechou
	if cb.state != StateClosed {
		t.Errorf("Estado deveria ter voltado para Closed após sucesso no Half-Open, é %v", cb.state)
	}

	// Verificar se contadores zeraram (uma falha agora não deve abrir)
	cb.Execute(failAction) 
	if cb.state == StateOpen {
		t.Error("Uma única falha após reset não deveria abrir o circuito imediatamente (assumindo threshold > 1)")
	}
}
