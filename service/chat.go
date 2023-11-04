package service

import (
	"golang.org/x/net/websocket"
	"fmt"
	"encoding/json"
	"time"
	"hash/fnv"
)

type chatacc struct {
	id int
	hash HashResult
	name string
	email string
	connected bool
}

type chatmsg struct {
	Id float64 `json:"id"`
	Hash float64 `json:"hash"`
	Action string `json:"action"`
	Info map[string]any `json:"info"`
}

var chatUsers = make(map[*websocket.Conn]*chatacc)

func accExecute(ws *websocket.Conn, msg chatmsg) {
	fmt.Printf("%+v\n", msg)
	if (msg.Action == "set-username") {
		fmt.Println(msg.Info)
		chatUsers[ws].email = msg.Info["email"].(string)
		chatUsers[ws].name = msg.Info["name"].(string)
	} else if (msg.Action == "message") {

		broadcast( map[string]any{
			"action": "user-msg",
			"from": chatUsers[ws].name,
			"email": chatUsers[ws].email,
			"msg": msg.Info["msg"],
		}, ws)
	}
}

func accParse(ws *websocket.Conn, read []byte) (e error) {
	var msg chatmsg
	e = json.Unmarshal(read, &msg)
	if (e != nil) {
		return e
	}
	accExecute(ws, msg)
	return nil
}

func accReadLoop(ws *websocket.Conn) error {
	buf := make([]byte, 1024)
	for {
		n, err := ws.Read(buf)
		if (err != nil) {return nil} //EOF
		err = accParse(ws, buf[:n])
		if ( err != nil) {return err}
	}
}

// Echo the data received on the WebSocket.
func chatServer(ws *websocket.Conn) {
	id := len(chatUsers)
	hash := Hash(time.Now().String())
	chatUsers[ws] = &chatacc{id, hash, "", "", true}

	dt, e := json.Marshal(map[string]any{"id":id, "hash":hash})
	if (e != nil) {panic(e)}
	ws.Write(dt)

	e = accReadLoop(ws)
	if (e != nil) { panic(e) }
	chatUsers[ws].connected = false

	broadcast( map[string]any{
		"action": "server-msg",
		"from": chatUsers[ws].name,
		"email": chatUsers[ws].email,
		"msg": chatUsers[ws].name+" disconnected",
	}, nil)
	//dt, e = json.Marshal(map[string]any{
	//	"action": "server-msg",
	//	"from": chatUsers[ws].name,
	//	"email": chatUsers[ws].email,
	//	"msg": chatUsers[ws].name+" disconnected",
	//})
	//if (e != nil) {panic(e)}

	//for otherws, acc := range chatUsers {
	//	if (acc.connected) {
	//		otherws.Write(dt)
	//	}
	//}
}

func broadcast(info map[string]any, sender *websocket.Conn) {
	dt, e := json.Marshal(info)
	if (e != nil) {panic(e)}

	for otherws, acc := range chatUsers {
		if (acc.connected && otherws != sender) {
			otherws.Write(dt)
		}
	}
}

var ChatServer = websocket.Handler(chatServer)

type HashResult = uint32
const HashBitLen = 32
func Hash(s string) HashResult {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()+uint32(90749*len(s))
}

