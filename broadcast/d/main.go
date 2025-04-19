package main

import (
	"encoding/json"
	"log"
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
	messages     []int
	batch        []int
	messageIndex map[int]int

	timer *time.Timer
	mu    *sync.Mutex

	node *maelstrom.Node
}

func (e *Env) broadcast(messages []int) {
	for _, nodeID := range e.node.NodeIDs() {
		if nodeID == e.node.ID() {
			continue
		}

		go func(nodeID string) {
			body := struct {
				Type     string `json:"type"`
				Messages []int  `json:"messages"`
			}{
				Type:     "message",
				Messages: messages,
			}

			acked := false
			timeout := 400
			retries := 0
			MAX_RETRIES := 10

			for !acked {
				retries++

				err := e.node.RPC(nodeID, body, func(message maelstrom.Message) error {
					acked = true

					return nil
				})

				if retries > MAX_RETRIES {
					log.Printf("[broadcast] reached max retry for msg '%d' from '%s' to %s", messages, e.node.ID(), nodeID)
					return
				}

				if err != nil {
					log.Printf("[broadcast] failed sending msg '%d' or node '%s'", messages, nodeID)
				}

				time.Sleep(time.Millisecond * time.Duration(timeout))
				timeout *= 2
			}
		}(nodeID)
	}
}

const (
	batchSize  = 3
	flushDelay = 100 * time.Millisecond
)

func (e *Env) flush() {
	e.broadcast(e.batch)

	e.mu.Lock()
	e.batch = make([]int, 0)
	e.mu.Unlock()
}

func (e *Env) batchMessage(msg int) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.batch = append(e.batch, msg)

	if len(e.batch) == 1 {
		e.timer = time.AfterFunc(flushDelay, e.flush)

		return
	}

	if len(e.batch) >= batchSize {
		if e.timer != nil {
			e.timer.Stop()
		}

		go e.flush()
	}
}

func main() {
	node := maelstrom.NewNode()
	var mu sync.Mutex

	env := Env{
		node: node,

		messageIndex: make(map[int]int, 0),
		messages:     make([]int, 0),
		batch:        make([]int, 0),

		mu: &mu,
	}

	// broadcast messages on the cluster
	env.node.Handle("broadcast", func(msg maelstrom.Message) error {
		var body BroadcastRequest

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		res := BroadcastResponse{
			Type: "broadcast_ok",
		}

		mu.Lock()
		if _, ok := env.messageIndex[body.Message]; !ok {
			env.messageIndex[body.Message] = body.Message
			env.messages = append(env.messages, body.Message)

			go env.batchMessage(body.Message)
		}
		mu.Unlock()

		return env.node.Reply(msg, res)
	})

	// receive broadcasted value
	env.node.Handle("message", func(msg maelstrom.Message) error {
		var body struct {
			Type     string `json:"type"`
			Messages []int  `json:"messages"`
		}

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		mu.Lock()
		env.messages = append(env.messages, body.Messages...)
		mu.Unlock()

		return env.node.Reply(msg, struct {
			Type string `json:"type"`
		}{Type: "message_ok"})
	})

	// cluster topology
	env.node.Handle("topology", func(msg maelstrom.Message) error {
		// ignore topology

		return env.node.Reply(msg, TopologyResponse{
			Type: "topology_ok",
		})
	})

	// read
	env.node.Handle("read", func(msg maelstrom.Message) error {
		return env.node.Reply(msg, ReadResponse{
			Type:     "read_ok",
			Messages: env.messages,
		})
	})

	// acknowledge broadcasted messages
	env.node.Handle("broadcast_ok", func(msg maelstrom.Message) error {
		return nil
	})

	if err := env.node.Run(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
