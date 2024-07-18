package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
)

var upgrader = ws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type group struct {
	host   *ws.Conn
	peers  []*ws.Conn
	banner []byte
}

var store = map[string]*group{}

func main() {
	r := gin.Default()

	register_socket(r)
	r.Run()
}

func register_socket(r *gin.Engine) {
	r.GET("/", func(ctx *gin.Context) {
		conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "connection upgrade error: %s\n", err.Error())
			return
		}
		defer conn.Close()

		id := uuid.New().String()
		store[id] = &group{host: conn}

		conn.WriteMessage(ws.TextMessage, []byte(id))

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Fprintf(os.Stderr, "read from connection error: %s\n", err.Error())
				return
			}

			if handle_msg(msg, conn, id) {
				break
			}
		}
	})
}

func handle_msg(msg []byte, conn *ws.Conn, id string) (exit bool) {
	if string(msg) == "exit" {
		return true
	}

	cmd, rest, _ := strings.Cut(string(msg), " ")
	switch cmd {
	case "join": // cmd: `join <id>` where `id` is that of the group to join
		if group, ok := store[rest]; ok {
			group.peers = append(group.peers, conn)
			conn.WriteMessage(ws.TextMessage, group.banner)
		} else {
			conn.WriteMessage(ws.TextMessage, []byte("no group found"))
		}
	case "write": // cmd: `write <msg>` where `msg` is sent to the all the
		// peers in your group
		if self, ok := store[id]; ok {
			for _, peer := range self.peers {
				peer.WriteMessage(ws.TextMessage, []byte(rest))
			}
		} else {
			conn.WriteMessage(ws.TextMessage, []byte("no group created"))
		}
	case "add": // cmd: `add <id>` where `id` is that of the connection that
		// is to be added to your group
		if other, ok := store[rest]; ok {
			if self, ok := store[id]; ok {
				self.peers = append(self.peers, other.host)
				conn.WriteMessage(ws.TextMessage, []byte("added to self"))
				other.host.WriteMessage(ws.TextMessage, self.banner)
			}
		}
	case "banner": // cmd: `banner <info>` where `info` is the details to be
		// set on the banner which will be writen to the first joinee to your group
		if self, ok := store[id]; ok {
			self.banner = []byte(rest)
		}
	}

	return false
}
