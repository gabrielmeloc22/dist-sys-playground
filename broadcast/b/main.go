package main

import (
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"slices"
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

type TopologyRequest struct {
	Type     string              `json:"type"`
	MsgID    int                 `json:"msg_id"`
	Topology map[string][]string `json:"topology"`
}

type TopologyResponse struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
}

func broadcast(msg int, neighbors []string, n *maelstrom.Node) error {

	for _, nodeID := range neighbors {
		if nodeID == n.ID() {
			continue
		}

		go func() {
			body := struct {
				Type string `json:"type"`
				Msg  int    `json:"message"`
			}{
				Type: "broadcast",
				Msg:  msg,
			}

			n.Send(nodeID, body)

		}()
	}

	return nil
}

func main() {
	n := maelstrom.NewNode()

	messages := make([]int, 0)
	neighbors := make([]string, 0)

	// cluster topology
	n.Handle("topology", func(msg maelstrom.Message) error {
		var body TopologyRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		neighbors = body.Topology[n.ID()]

		return n.Reply(msg, TopologyResponse{
			MsgID: body.MsgID,
			Type:  "topology_ok",
		})
	})

	// read
	n.Handle("read", func(msg maelstrom.Message) error {
		var body ReadRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		return n.Reply(msg, ReadResponse{
			MsgID:    body.MsgID,
			Type:     "read_ok",
			Messages: messages,
		})
	})

	// acknowledge broadcasted messages
	n.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		return nil
	})

	// broadcast messages on the cluster
	n.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		res := BroadcastResponse{
			MsgID: body.MsgID,
			Type:  "broadcast_ok",
		}

		if slices.Contains(messages, body.Message) {
			return n.Reply(msg, res)
		}

		messages = append(messages, body.Message)

		if err := broadcast(body.Message, neighbors, n); err != nil {
			return err
		}

		return n.Reply(msg, res)
	})

	if err := n.Run(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
