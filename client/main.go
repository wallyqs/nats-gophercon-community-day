package main

import (
	"log"
	"time"
)

func main() {
	c := &Client{}
	err := c.Connect("127.0.0.1:4222")
	if err != nil {
		log.Fatalf("Error: %s", err)
	}
	defer c.Close()

	// Subscribing to a couple of subjects
	c.Subscribe("foo", "", func(subject, reply string, data []byte) {
		log.Println("[SUB ] -", subject, reply, "->", string(data))
	})
	c.Subscribe("hello", "", func(subject, reply string, data []byte) {
		log.Println("[SUB ] -", subject, reply, "->", string(data))
	})

	for {
		// Publishing a couple of commands.
		c.Publish("foo", "", []byte("bar"))
		c.Publish("hello", "", []byte("world"))
		c.Flush()
		time.Sleep(1 * time.Second)
	}
}
