package service

import (
	"golang.org/x/net/websocket"
	"sync"
	"io"
	"fmt"
	//"encoding/json"
)

type AVec[T any] struct {
	data []T
	lock sync.Mutex
}

func (V *AVec[T]) Append(value T) {
	V.lock.Lock()
	defer V.lock.Unlock()

	V.data = append(V.data, value)
}

func (V *AVec[T]) Map(fn func(T)any) []any {
	V.lock.Lock()
	defer V.lock.Unlock()

	var ret = make([]any, len(V.data))
	for i, item :=range V.data {
		ret[i] = fn(item)
	}

	return ret
}

func (V *AVec[T]) Acquire() *[]T {
	V.lock.Lock()
	return &V.data
}

func (V *AVec[T]) Release() {
	V.lock.Unlock()
}

var chatUsers = AVec[*websocket.Conn]{}

// Echo the data received on the WebSocket.
func chatServer(ws *websocket.Conn) {
	chatUsers.Append(ws)
	fmt.Println("hi")
	for {
		bts, n := io.ReadAll(ws)
		fmt.Printf("read %d bytes %q\n", n, bts)
		ws.Write(bts)
	}
}

var ChatServer = websocket.Handler(chatServer)
