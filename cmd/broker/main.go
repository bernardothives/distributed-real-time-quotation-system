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

func (b *Broker) Unsubscribe(topic string, conn net.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subscribers, ok := b.subscribers[topic]
	if !ok {
		return
	}

	for i, c := range subscribers {
		if c == conn {
			// Safe removal preserving order (optional) or fast removal
			b.subscribers[topic] = append(subscribers[:i], subscribers[i+1:]...)
			fmt.Printf("Removed subscriber from topic: %s\n", topic)
			// Close connection to be sure
			conn.Close()
			return
		}
	}
}

func (b *Broker) Publish(topic string, msg protocol.Message) {
	// Critical: Copy the slice to avoid holding RLock during IO or Data Races
	// if Unsubscribe changes the underlying array while we iterate.
	b.mu.RLock()
	originalConns := b.subscribers[topic]
	conns := make([]net.Conn, len(originalConns))
	copy(conns, originalConns)
	b.mu.RUnlock()

	if len(conns) == 0 {
		return
	}

	fmt.Printf("Broadcasting to %d subscribers on topic %s\n", len(conns), topic)

	for _, conn := range conns {
		go func(c net.Conn) {
			// Check for error on send
			if err := protocol.SendJSON(c, msg); err != nil {
				fmt.Printf("Error sending to subscriber: %v. Removing.\n", err)
				b.Unsubscribe(topic, c)
			}
		}(conn)
	}
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
	// Only close at the very end of the session
	defer conn.Close()

	for {
		var msg protocol.Message
		if err := protocol.ReceiveJSON(conn, &msg); err != nil {
			// If we can't read, the client is gone.
			// Note: Ideally we should cleanup subscriptions here too if they subscribed,
			// but for this simple architecture, the Write failure in Publish will clean it up eventually.
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