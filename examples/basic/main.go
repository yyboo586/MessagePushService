package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	testWebSocket()
}

func testWebSocket() {
	var wg sync.WaitGroup

	for i := 1; i <= 5000; i++ {
		userID := fmt.Sprintf("%d", i)
		c, _, err := websocket.DefaultDialer.Dial("ws://localhost:9701/ws/private?user_id="+userID, nil)
		if err != nil {
			log.Fatal("dial:", err)
		}

		c.SetCloseHandler(func(code int, text string) error {
			return &websocket.CloseError{Code: code, Text: text}
		})

		wg.Add(2)
		exitChan := make(chan struct{}, 1)
		go read(c, &wg, exitChan)
		go write(c, &wg, exitChan)

	}

	wg.Wait()
}

func read(c *websocket.Conn, wg *sync.WaitGroup, exitchan chan struct{}) {
	defer wg.Done()

	for {
		select {
		case <-exitchan:
			return
		default:
			messageType, message, err := c.ReadMessage()
			if err != nil {
				if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
					log.Println("close:", err)
				} else {
					log.Println("read:", err)
				}
				return
			}
			log.Printf("recv: %d, %s", messageType, message)
		}
	}
}

func write(c *websocket.Conn, wg *sync.WaitGroup, exitChan chan struct{}) {
	defer wg.Done()

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for i := 1; i <= 50; i++ {
		err := c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("hello %d", i)))
		if err != nil {
			log.Fatal("write:", err)
		}
	}
	c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))

	exitChan <- struct{}{}
}
