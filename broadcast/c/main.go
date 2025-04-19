package main

import (
	"encoding/json"
	"log"
	"slices"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type BroadcastRequest struct {
	MsgID   int    `json:"msg_id"`
	Type    string `json:"type"`
	Message int    `json:"message"`
}

type BroadcastResponse struct {
	Type string `json:"type"`
}

type ReadResponse struct {
	Type     string `json:"type"`
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
	Type string `json:"type"`
}

type BroadcastOkRequest struct {
	Type  string `json:"type"`
	MsgID int    `json:"msg_id"`
}

type Env struct {
	messages  []int
	neighbors []string

	node *maelstrom.Node
}

func (e *Env) broadcast(msg int, msgId int) {
	for _, nodeID := range e.neighbors {
		if nodeID == e.node.ID() {
			continue
		}

		go func(nodeID string) {
			body := struct {
				Type string `json:"type"`
				Msg  int    `json:"message"`
			}{
				Type: "broadcast",
				Msg:  msg,
			}

			acked := false
			timeout := 100

			for !acked {
				err := e.node.RPC(nodeID, body, func(message maelstrom.Message) error {
					acked = true

					return nil
				})

				if err != nil {
					log.Printf("[broadcast] failed sending msg '%d' or node '%s'", msgId, nodeID)
				}

				time.Sleep(time.Millisecond * time.Duration(timeout))
			}
		}(nodeID)
	}
}

func main() {
	var mu sync.Mutex

	env := Env{
		node: maelstrom.NewNode(),

		messages:  make([]int, 0),
		neighbors: make([]string, 0),
	}

	// cluster topology
	env.node.Handle("topology", func(msg maelstrom.Message) error {
		var body TopologyRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		log.Printf("[topology] setting topology \n%s", msg.Body)

		env.neighbors = body.Topology[env.node.ID()]

		return env.node.Reply(msg, TopologyResponse{
			Type: "topology_ok",
		})
	})

	// read
	env.node.Handle("read", func(msg maelstrom.Message) error {
		var body ReadRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			log.Printf("[read] error parsing body:\n%s", msg.Body)
			return err
		}

		mu.Lock()
		defer mu.Unlock()

		return env.node.Reply(msg, ReadResponse{
			Type:     "read_ok",
			Messages: env.messages,
		})
	})

	// acknowledge broadcasted messages
	env.node.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		var body BroadcastOkRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			log.Printf("[broadcast_ok] error parsing body:\n%s", msg.Body)
			return err
		}

		log.Printf(" [%s] got ACK for msg %d from %s", env.node.ID(), body.MsgID, msg.Src)

		return nil
	})

	// broadcast messages on the cluster
	env.node.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			log.Printf("[broadcast] error parsing body:\n%s", msg.Body)
			return err
		}

		res := BroadcastResponse{
			Type: "broadcast_ok",
		}

		mu.Lock()

		if !slices.Contains(env.messages, body.Message) {
			env.messages = append(env.messages, body.Message)

			env.broadcast(body.Message, body.MsgID)
		}
		mu.Unlock()

		return env.node.Reply(msg, res)
	})

	if err := env.node.Run(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
