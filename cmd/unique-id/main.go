package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"syscall"
	"time"

	"os"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func logError(err error) {
	log.Fatalf("ERROR: %s", err)
}

type DistributedLock struct {
	lock *os.File
}

func (c *DistributedLock) Lock(sleep time.Duration, retry int) error {
	count := 0

	lockFile, err := os.Open("lock")

	if err != nil && os.IsNotExist(err) {
		newLockFile, err := os.Create("lock")

		if err != nil {
			return err
		}

		lockFile = newLockFile
	}

	c.lock = lockFile

	for count < retry {
		if err := syscall.Flock(int(c.lock.Fd()), syscall.LOCK_EX); err != nil {
			break
		}

		count++
		time.Sleep(sleep)
	}

	if count == retry {
		return errors.New("could not acquire lock")
	}

	return nil
}

func (c *DistributedLock) Unlock() error {
	if c.lock == nil {
		return errors.New("lock not initialized")
	}

	err := syscall.Flock(int(c.lock.Fd()), syscall.LOCK_UN)

	if err != nil {
		return err
	}

	return nil
}

func generateId() int {
	l := DistributedLock{}

	l.Lock(time.Duration(time.Nanosecond), 10)
	defer l.Unlock()

	if _, err := os.Stat("db.txt"); errors.Is(err, os.ErrNotExist) {

		file, err := os.Create("db.txt")
		file.Write([]byte("0"))

		if err != nil {
			logError(err)
		}
	}

	db, err := os.OpenFile("db.txt", os.O_RDWR, os.ModePerm)

	if err != nil {
		logError(err)
	}

	buff := make([]byte, 1024)
	if _, err := db.Read(buff); err != nil {
		logError(err)
	}

	lastId, err := strconv.Atoi(string(bytes.Trim(buff, "\x00")))

	if err != nil {
		logError(err)
	}

	newId := lastId + 1

	if _, err = db.WriteAt([]byte(fmt.Sprintf("%d", newId)), 0); err != nil {
		logError(err)
	}

	return newId
}

func main() {
	n := maelstrom.NewNode()

	n.Handle("generate", func(msg maelstrom.Message) error {
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		body["type"] = "generate_ok"
		body["id"] = generateId()

		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Fatalf("ERROR: %s", err)
	}
}
