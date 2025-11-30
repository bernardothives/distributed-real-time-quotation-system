# Sistema de Cotações em Tempo Real Distribuído

**Disciplina:** INE5645 - Programação Paralela e Distribuída  
**Linguagem:** Go (Golang) 1.21+  
**Relatório Técnico:** Veja [RELATORIO.md](RELATORIO.md) para detalhes arquiteturais e justificativas teóricas.

---

## Visão Geral

Este projeto implementa uma infraestrutura de microsserviços para simular um sistema financeiro de alta disponibilidade. A solução demonstra a aplicação prática de quatro padrões fundamentais de sistemas distribuídos para resolver problemas de latência, acoplamento e tolerância a falhas.

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

## Como Executar

### Pré-requisitos
*   **Go** (versão 1.20 ou superior)
*   **Make** (para automação de build)
*   Sistema Operacional Linux ou macOS (devido ao uso de sinais de processo no Makefile)

### 1. Compilação
Compila todos os microsserviços e gera os binários na pasta `bin/`.
```bash
make build
```

### 2. Inicialização (Infraestrutura)
Inicia todos os serviços em background (`&`), salvando os PIDs para encerramento posterior.
```bash
make run-all
```
> *Aguarde mensagem "All services started."*

### 3. Teste: Pub/Sub (Tempo Real)
Inicia um cliente que se subscreve no Broker. Você verá atualizações de preço chegando via "push" assim que o Core processar cotações.
```bash
make test-sub
```
*(Pressione Ctrl+C para sair)*

### 4. Teste: Scatter/Gather (Relatório Agregado)
Solicita um relatório completo. O Aggregator buscará o preço atual no Core e o histórico nos 3 Shards simultaneamente.
```bash
make test-aggregator
```

### 5. Parar o Sistema
Encerra todos os processos (kill) e remove arquivos temporários (`.pid`).
```bash
make stop-all
```

---

## Estrutura de Diretórios

*   `cmd/`: Ponto de entrada (main) de cada microsserviço.
*   `pkg/`: Bibliotecas compartilhadas (Protocolo, Modelos, Circuit Breaker).
*   `bin/`: Binários executáveis (ignorados pelo git).