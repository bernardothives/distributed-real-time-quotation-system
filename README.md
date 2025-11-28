# Sistema Distribuído de Cotações em Tempo Real
**Disciplina:** INE5645 - Padrões de Projeto para Sistemas Distribuídos  
**Linguagem:** Go (Golang)

## 1. Justificativa Técnica dos Padrões

### 1.1 Sharding (Particionamento)
**Problema Teórico:** Bancos de dados monolíticos tornam-se gargalos de performance (IOPS e Memória) conforme o histórico cresce.
**Solução:** Utilizamos Sharding para particionar horizontalmente os dados históricos. Implementamos 3 processos independentes (`HistoryShard`) operando em portas distintas.
**Benefício na Solução:** O `AggregatorService` distribui as consultas. Se o critério fosse chave (Symbol), a carga seria balanceada. Como implementamos *sharding* por partição de dados (simulado aqui), permitimos que o sistema escale linearmente: para dobrar a capacidade de armazenamento ou throughput de leitura, basta adicionar mais processos Shards sem alterar a lógica do Core.

### 1.2 Pub/Sub (Publisher/Subscriber)
**Problema Teórico:** Acoplamento direto entre o gerador de preços (Core) e os múltiplos clientes interessados impediria a escalabilidade. O Core travaria tentando enviar dados serialmente para milhares de clientes.
**Solução:** Implementamos um `BrokerService`. O `CoreService` (Publisher) envia a mensagem apenas uma vez para o Broker. O Broker mantém um mapa de Tópicos e Sockets.
**Benefício na Solução:** Desacoplamento **Espacial** (Core não conhece Clientes) e **Temporal** (o processamento de envio para N clientes é feito pelo Broker, liberando o Core para processar a próxima cotação imediatamente).

### 1.3 Circuit Breaker
**Problema Teórico:** O `ExternalQuoteService` é instável. Falhas temporárias ou latência alta podem travar todas as threads do nosso `CoreService` (esgotamento de recursos), derrubando o sistema todo (Falha em Cascata).
**Solução:** Implementamos uma máquina de estados (Closed, Open, Half-Open) no pacote `pkg/circuitbreaker`.
**Benefício na Solução:** Se o serviço externo falhar repetidamente (threshold configurado), o Circuit Breaker "abre" e falha imediatamente as requisições subsequentes sem tentar conexão de rede. Isso preserva CPU/Threads do Core e dá tempo para o serviço externo se recuperar (Time-based sliding window).

### 1.4 Scatter/Gather
**Problema Teórico:** O cliente precisa de um relatório complexo: Preço Atual (origem A) + Histórico (origens B, C, D). Fazer isso sequencialmente (A -> B -> C -> D) somaria as latências (Latência = T_a + T_b + T_c...).
**Solução:** O `AggregatorService` dispara goroutines em paralelo para todas as fontes.
**Benefício na Solução:** A latência final da requisição é determinada pela resposta *mais lenta* do conjunto, e não pela soma de todas (`max(T_a, T_b...)`). Usamos `sync.WaitGroup` para sincronizar o "Gather" (união dos resultados) antes de devolver o JSON ao cliente.

## 2. Como Executar

### Pré-requisitos
- Go 1.20+ instalado.
- Ambiente Linux/Mac (para uso do Makefile e sinais de sistema).

### Passo a Passo
1. **Compilar e Iniciar Infraestrutura:**
   Isso iniciará 7 processos: External, Broker, Core, 3x Shards e Aggregator.
   ```bash
   make run-all
   ```

2. **Testar Cliente (Subscriber):**
   Em outro terminal, abra um assinante que ficará ouvindo atualizações em tempo real.
   ```bash
   make test-sub
   ```

3. **Testar Cliente (Aggregator/Relatório):**
   Em um terceiro terminal, peça o relatório completo. Isso disparará o Scatter/Gather e forçará o Core a buscar cotação no External (podendo ativar o Circuit Breaker se houver falha simulada).
   ```bash
   make test-aggregator
   ```

4. **Parar Tudo:**
   ```bash
   make stop-all
   ```
