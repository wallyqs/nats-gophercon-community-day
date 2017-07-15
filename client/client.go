package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type Client struct {
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer

	// sid increments monotonically per subscription and it is
	// used to identify a subscription from the client when
	// receiving a message.
	sid int

	// subs maps a subscription identifier to a callback.
	subs map[int]func(subject, reply string, b []byte)

	// pongs is a channel used to signal whenever a pong is received.
	pongs chan (struct{})

	sync.Mutex
}

func (c *Client) Connect(netloc string) error {
	conn, err := net.Dial("tcp", netloc)
	if err != nil {
		return err
	}
	c.conn = conn
	c.r = bufio.NewReader(conn)
	c.w = bufio.NewWriter(conn)
	c.subs = make(map[int]func(string, string, []byte))
	c.pongs = make(chan struct{}, 2)

	connect := struct {
		Name    string `json:"name"`
		Verbose bool   `json:"verbose"`
	}{
		Name:    "gopher",
		Verbose: false,
	}
	connectOp, err := json.Marshal(connect)
	if err != nil {
		return err
	}
	connectCmd := fmt.Sprintf("CONNECT %s\r\n", connectOp)
	_, err = c.w.WriteString(connectCmd)
	if err != nil {
		return err
	}

	err = c.w.Flush()
	if err != nil {
		return err
	}

	// Spawn goroutine for the parser reading loop.
	go c.runParserLoop()

	return nil
}

func (c *Client) Close() {
	// Send any pending commands to server previous
	// to closing.
	c.w.Flush()
	c.conn.Close()
}

func (c *Client) Publish(subject, reply string, payload []byte) error {
	c.Lock()
	defer c.Unlock()

	pub := fmt.Sprintf("PUB %s %s %d\r\n", subject, reply, len(payload))
	_, err := c.w.WriteString(pub)
	if err != nil {
		return err
	}
	_, err = c.w.Write(payload)
	if err != nil {
		return err
	}
	_, err = c.w.WriteString("\r\n")
	if err != nil {
		return err
	}
	err = c.w.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Subscribe(subject, queue string, cb func(subject, reply string, b []byte)) error {
	c.Lock()
	defer c.Unlock()
	c.sid += 1
	sid := c.sid

	sub := fmt.Sprintf("SUB %s %s %d\r\n", subject, queue, sid)
	_, err := c.w.WriteString(sub)
	if err != nil {
		return err
	}

	err = c.w.Flush()
	if err != nil {
		return err
	}

	c.subs[sid] = cb

	return nil
}

func (c *Client) Flush() error {
	_, err := c.w.WriteString("PING\r\n")
	if err != nil {
		return err
	}
	err = c.w.Flush()
	if err != nil {
		return err
	}

	c.Lock()
	pongs := c.pongs
	c.Unlock()

	if pongs == nil {
		return fmt.Errorf("nats: invalid pongs channel")
	}

	// Wait for a PONG back to be received
	// before continuing.
	select {
	case <-c.pongs:
	case <-time.After(5 * time.Second):
		return fmt.Errorf("nats: flush timeout")
	}

	return nil
}

func (c *Client) processInfo(line string) {
	info := struct {
		MaxPayload  int      `json:"max_payload"`
		ConnectUrls []string `json:"connect_urls"`
	}{}
	json.Unmarshal([]byte(line), &info)

	log.Printf("[INFO] - %+v", info)
}

func (c *Client) processMsg(subj string, reply string, sid int, payload []byte) {
	log.Printf("[MSG ] - subject=%q, reply=%q, payload=%v",
		subj, reply, string(payload))
	c.Lock()
	cb, ok := c.subs[sid]
	c.Unlock()

	if ok {
		// Problem: This would block the parser read loop.
		cb(subj, reply, payload)
	}
}

func (c *Client) processPing() {
	log.Printf("[PING]")

	// Reply back to prevent stale connection error.
	c.w.WriteString("PONG\r\n")
	c.w.Flush()
}

func (c *Client) processPong() {
	log.Printf("[PONG]")

	c.Lock()
	pongs := c.pongs
	c.Unlock()

	// Problem: We are signaling on any pong received
	// rather on the ping/pong we scheduled.
	if pongs != nil {
		pongs <- struct{}{}
	}
}

func (c *Client) processErr(msg string) {
	log.Printf("[-ERR] - %s", msg)
}

func (c *Client) runParserLoop() {
	for {
		line, err := c.r.ReadString('\n')
		if err != nil {
			log.Fatalf("Error: %s", err)
		}
		args := strings.SplitN(line, " ", 2)
		if len(args) < 1 {
			log.Fatalf("Error: malformed control line")
		}

		op := strings.TrimSpace(args[0])
		switch op {
		case "MSG":
			var subject, reply string
			var sid, size int

			n := strings.Count(args[1], " ") + 1
			switch n {
			case 3:
				// No reply inbox case.
				// MSG foo 1 3\r\n
				// bar\r\n
				_, err := fmt.Sscanf(args[1], "%s %d %d", &subject, &sid, &size)
				if err != nil {
					log.Fatalf("Error: malformed control line: %s", err)
				}
			case 4:
				// With reply inbox case.
				// MSG foo 1 bar 4\r\n
				// quux\r\n
				_, err := fmt.Sscanf(args[1], "%s %d %s %d", &subject, &sid, &reply, &size)
				if err != nil {
					log.Fatalf("Error: malformed control line: %s", err)
				}
			default:
				log.Fatalf("Error: malformed control line")
			}

			// Prepare buffer for the payload
			payload := make([]byte, size)
			_, err = io.ReadFull(c.r, payload)
			if err != nil {
				log.Fatalf("Error: problem gathering bytes: %s", err)
			}
			c.processMsg(subject, reply, sid, payload)
		case "INFO":
			c.processInfo(args[1])
		case "PING":
			c.processPing()
		case "PONG":
			c.processPong()
		case "+OK":
			// Do nothing.
		case "-ERR":
			c.processErr(args[1])
		}
	}
}
