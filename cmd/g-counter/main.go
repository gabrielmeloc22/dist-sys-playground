package main

import (
	"context"
	"encoding/json"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

var COUNTER_KEY = "counter"
var kv *maelstrom.KV

func read(ctx context.Context) (int, error) {
	value, err := kv.Read(ctx, COUNTER_KEY)

	if err != nil {
		return 0, err
	}

	return value.(int), nil
}

func write(ctx context.Context, value int) error {
	count, err := read(ctx)

	if rpcErr, ok := err.(*maelstrom.RPCError); ok && rpcErr.Code == maelstrom.KeyDoesNotExist {
		count = 0
	}

	accepted := false

	for !accepted {
		err = kv.CompareAndSwap(ctx, COUNTER_KEY, count, count+value, true)

		if rpcErr, ok := err.(*maelstrom.RPCError); ok && rpcErr.Code != maelstrom.PreconditionFailed {
			return err
		}

		if err == nil {
			accepted = true
		}
	}

	return nil
}

func main() {
	n := maelstrom.NewNode()
	kv = maelstrom.NewSeqKV(n)
	ctx := context.Background()

	n.Handle("add", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		var getSuccessBody = func() map[string]any {
			body["type"] = "add_ok"
			delete(body, "delta")

			return body
		}

		delta := int(body["delta"].(float64))

		write(ctx, delta)

		return n.Reply(msg, getSuccessBody())
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		value, err := read(ctx)

		if err != nil {
			value = 0
		}

		body["type"] = "read_ok"
		body["value"] = value

		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
