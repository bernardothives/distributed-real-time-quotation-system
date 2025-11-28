package main

import (
	"distributed-system/pkg/protocol"
	"fmt"
	"net"
	"sync"
)

type Broker struct {
	subscribers map[string][]net.Conn
	mu          sync.RWMutex
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[string][]net.Conn),
	}
}

func (b *Broker) Subscribe(topic string, conn net.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[topic] = append(b.subscribers[topic], conn)
	fmt.Printf("New subscriber for topic: %s\n", topic)
}

func (b *Broker) Publish(topic string, msg protocol.Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	conns := b.subscribers[topic]
	for _, conn := range conns {
		// In a real system, we would handle slow consumers/disconnects here
		go protocol.SendJSON(conn, msg)
	}
	fmt.Printf("Published message to %d subscribers on topic %s\n", len(conns), topic)
}

func main() {
	broker := NewBroker()
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("Broker Service running on :8081")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn, broker)
	}
}

func handleClient(conn net.Conn, broker *Broker) {
	// Not deferring close here immediately, depends on logic, but usually yes.
	// However, for subscribers, we want to keep connection open.
	// If read fails, we close.
	defer conn.Close()

	for {
		var msg protocol.Message
		if err := protocol.ReceiveJSON(conn, &msg); err != nil {
			return
		}

		switch msg.Type {
		case protocol.MsgSubscribe:
			broker.Subscribe(msg.Topic, conn)
		case protocol.MsgPublish:
			broker.Publish(msg.Topic, msg)
		}
	}
}
