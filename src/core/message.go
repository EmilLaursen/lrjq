package core

import "encoding/json"

type Message struct {
	Headers  json.RawMessage
	Payload  json.RawMessage
	Priority int32
	QueueID  string
}
