package main

import (
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastRequest struct {
	Type    string `json:"type"`
	Message int    `json:"message"`
}

type BroadcastResponse struct {
	Type string `json:"type"`
}

const BroadcastResponseType = "broadcast_ok"

func main() {
	n := maelstrom.NewNode()

	n.Handle("broadcast", func(msg maelstrom.Message) error {

		var body BroadcastRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		return n.Reply(msg, BroadcastResponse{Type: BroadcastResponseType})
	})
}
