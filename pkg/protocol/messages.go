package protocol

import (
	"encoding/json"
	"net"
)

// Message Types
const (
	MsgSubscribe    = "SUBSCRIBE"
	MsgPublish      = "PUBLISH"
	MsgRequestQuote = "REQ_QUOTE"
	MsgReqHistory   = "REQ_HIST"
	MsgRespQuote    = "RESP_QUOTE"
	MsgRespHistory  = "RESP_HIST"
	MsgError        = "ERROR"
)

type Message struct {
	Type    string          `json:"type"`
	Topic   string          `json:"topic,omitempty"`   // Used for Pub/Sub
	Payload json.RawMessage `json:"payload,omitempty"`
}

func NewMessage(msgType string, data interface{}) Message {
	payload, _ := json.Marshal(data)
	return Message{Type: msgType, Payload: payload}
}

func SendJSON(conn net.Conn, v interface{}) error {
	encoder := json.NewEncoder(conn)
	return encoder.Encode(v)
}

func ReceiveJSON(conn net.Conn, v interface{}) error {
	decoder := json.NewDecoder(conn)
	return decoder.Decode(v)
}
