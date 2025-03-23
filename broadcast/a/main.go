package main

import (
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastRequest struct {
	MsgID   int    `json:"msg_id"`
	Type    string `json:"type"`
	Message int    `json:"message"`
}

type BroadcastResponse struct {
	MsgID int    `json:"msg_id"`
	Type  string `json:"type"`
}

type ReadResponse struct {
	Type     string `json:"type"`
	MsgID    int    `json:"msg_id"`
	Messages []int  `json:"messages"`
}

type ReadRequest struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
}

const BroadcastResponseType = "broadcast_ok"

func main() {
	n := maelstrom.NewNode()

	messages := make([]int, 0)

	n.Handle("topology", func(msg maelstrom.Message) error {
		return n.Reply(msg, struct {
			Type string `json:"type"`
		}{
			Type: "topology_ok",
		})
	})

	n.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		return nil
	})

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		messages = append(messages, body.Message)

		return n.Reply(msg, BroadcastResponse{MsgID: body.MsgID, Type: BroadcastResponseType})
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var body ReadRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		return n.Reply(msg, ReadResponse{MsgID: body.MsgID, Type: "read_ok", Messages: messages})
	})

	if err := n.Run(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
