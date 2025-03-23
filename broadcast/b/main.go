package main

import (
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
	"slices"
)

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
		body := struct {
			MsgID    int                 `json:"msg_id"`
			Type     string              `json:"type"`
			Topology map[string][]string `json:"topology"`
		}{}

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		neighbors = body.Topology[n.ID()]

		res := struct {
			MsgID int    `json:"msg_id"`
			Type  string `json:"type"`
		}{
			MsgID: body.MsgID,
			Type:  "topology_ok",
		}

		return n.Reply(msg, res)
	})

	// read
	n.Handle("read", func(msg maelstrom.Message) error {
		body := struct {
			MsgID int    `json:"msg_id"`
			Type  string `json:"type"`
		}{}

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		res := struct {
			MsgID    int    `json:"msg_id"`
			Type     string `json:"type"`
			Messages []int  `json:"messages"`
		}{
			MsgID:    body.MsgID,
			Type:     "read_ok",
			Messages: messages,
		}

		return n.Reply(msg, res)
	})

	// acknowledge broadcasted messages
	n.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		return nil
	})

	// broadcast messages on the cluster
	n.Handle("broadcast", func(msg maelstrom.Message) error {
		body := struct {
			MsgID int    `json:"msg_id"`
			Type  string `json:"type"`
			Msg   int    `json:"message"`
		}{}

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		res := struct {
			MsgID int    `json:"msg_id"`
			Type  string `json:"type"`
		}{
			MsgID: body.MsgID,
			Type:  "broadcast_ok",
		}

		if slices.Contains(messages, body.Msg) {
			return n.Reply(msg, res)
		}

		messages = append(messages, body.Msg)

		if err := broadcast(body.Msg, neighbors, n); err != nil {
			return err
		}

		return n.Reply(msg, res)
	})

	if err := n.Run(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
