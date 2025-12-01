# Sistema Distribuído de Cotações em Tempo Real

Um sistema distribuído robusto simulando uma plataforma financeira de alta frequência. Este projeto demonstra a implementação "from scratch" (sem frameworks pesados) de padrões clássicos de design distribuído, focando em resiliência, escalabilidade e desacoplamento.

---

## Arquitetura e Padrões de Design

O sistema foi arquitetado para resolver problemas reais de engenharia de software distribuída:

### 1. Circuit Breaker
*   **Problema:** O serviço `External` (Bolsa de Valores) simula instabilidade e latência.
*   **Solução:** Implementação de uma máquina de estados (Closed, Open, Half-Open) no serviço `Core`.
*   **Benefício:** Impede falhas em cascata e protege o sistema de exaustão de recursos quando dependências externas falham.
*   **Localização:** `pkg/circuitbreaker`

### 2. Publish/Subscribe
*   **Problema:** Múltiplos clientes precisam de cotações em tempo real sem sobrecarregar o `Core`.
*   **Solução:** Um `Broker` TCP dedicado gerencia tópicos e assinaturas. O `Core` publica uma vez (Fan-out).
*   **Benefício:** O `Core` não conhece os consumidores finais; alta escalabilidade de leitura.
*   **Localização:** `cmd/broker` e `pkg/protocol`

### 3. Database Sharding
*   **Problema:** O volume de histórico de transações cresce indefinidamente.
*   **Solução:** Particionamento horizontal dos dados em 3 nós (`Shard A`, `Shard B`, `Shard C`).
*   **Benefício:** Distribuição de carga de I/O e armazenamento.
*   **Localização:** `cmd/shard`

### 4. Scatter/Gather
*   **Problema:** Clientes precisam de um relatório unificado (Preço Atual + Histórico Completo) vindo de fontes distintas.
*   **Solução:** O `Aggregator` dispara requisições paralelas para o `Core` e todos os `Shards`, aguardando (`Wait`) e combinando os resultados.
*   **Benefício:** Redução latência total (limitada pelo serviço mais lento, não pela soma).
*   **Localização:** `cmd/aggregator`

---

### Topologia do Sistema

A infraestrutura é composta por 7 processos distintos comunicando-se via TCP/JSON:

| Serviço | Porta TCP | Função | Padrão Associado |
| :--- | :--- | :--- | :--- |
| **External** | `:8080` | Mock de Bolsa de Valores (instável) | Fonte de Dados |
| **Broker** | `:8081` | Distribuição de mensagens (1:N) | **Pub/Sub** |
| **Core** | `:8082` | Lógica de Negócio Central | **Circuit Breaker** |
| **Shard A-C**| `:9001-03`| Armazenamento particionado | **Sharding** |
| **Aggregator**| `:8000` | Gateway de consulta unificada | **Scatter/Gather** |

---

### Pré-requisitos
*   **Go** 1.20+
*   **Make** (GNU Make)
*   Ambiente Linux/Unix

### Execução Rápida

O projeto utiliza um `Makefile` para orquestrar os 7 processos distribuídos simultaneamente.

1. **Subir a Infraestrutura:**
   Compila e inicia todos os serviços (External, Broker, Core, Shards, Aggregator) em background.
   ```bash
   make run-all
   ```

2. **Testar Fluxo Pub/Sub (Tempo Real):**
   Inicia um cliente assinante para visualizar o fluxo de cotações.
   ```bash
   make test-sub
   ```

3. **Testar Fluxo Scatter/Gather (Relatório):**
   Solicita a agregação de dados distribuídos.
   ```bash
   make test-aggregator
   ```

4. **Parar Tudo:**
   Mata os processos e limpa os arquivos `.pid`.
   ```bash
   make stop-all
   ```

---

## Qualidade e Testes

A robustez do sistema é garantida por uma suíte de testes automatizados cobrindo os componentes críticos.

Para executar a validação completa:
```bash
make test
```

### Cobertura dos Testes:
*   **Protocolo (`pkg/protocol`):** Valida a serialização/deserialização JSON e resiliência contra payloads corrompidos (Fuzzing básico).
*   **Circuit Breaker (`pkg/circuitbreaker`):** Teste de caixa branca da máquina de estados, garantindo transições corretas entre `Closed` -> `Open` -> `Half-Open` -> `Closed` baseadas em limiares de erro e timeouts.
*   **Aggregator Resilience (`cmd/aggregator`):** Mock servers validam se o agregador sobrevive à falha total ou parcial dos Shards (Connection Refused, Timeout).

---

## Estrutura do Projeto

```plaintext
/
├── bin/                 # Binários compilados (ignorados pelo git)
├── cmd/                 # Entrypoints dos microsserviços
│   ├── aggregator/      # Serviço de agregação (Scatter/Gather)
│   ├── broker/          # Servidor de Mensageria TCP
│   ├── client/          # Cliente CLI para testes manuais
│   ├── core/            # Regras de negócio e Circuit Breaker
│   ├── external/        # Simulador de API externa instável
│   └── shard/           # Nós de armazenamento (Sharding)
├── pkg/                 # Código compartilhado
│   ├── circuitbreaker/  # Lógica de proteção de falhas
│   ├── model/           # Entidades de Domínio (Quote, Transaction)
│   └── protocol/        # Protocolo de Comunicação Customizado (TCP/JSON)
├── Makefile             # Automação de build e testes
└── README.md            # Documentação
```
