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

type peer struct {
	uuid string
	conn *ws.Conn
}

type group struct {
	uuid   string
	host   *ws.Conn
	peers  []*peer
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
		self := &group{uuid: id, host: conn}
		store[id] = self

		conn.SetCloseHandler(clear_self(self))
		conn.WriteMessage(ws.TextMessage, []byte(id))

		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Fprintf(os.Stderr, "read from connection error: %s\n", err.Error())
				return
			}

			if handle_msg(msg, self) {
				break
			}
		}
	})
}

func handle_msg(msg []byte, self *group) (exit bool) {
	if string(msg) == "exit" {
		return true
	}

	cmd, rest, _ := strings.Cut(string(msg), " ")
	switch cmd {
	case "join": // cmd: `join <id>` where `id` is that of the other to join
		if other, ok := store[rest]; ok {
			other.peers = append(other.peers, &peer{uuid: self.uuid, conn: self.host})
			self.host.WriteMessage(ws.TextMessage, other.banner)
		} else {
			self.host.WriteMessage(ws.TextMessage, []byte("no group found"))
		}
	case "write": // cmd: `write <msg>` where `msg` is sent to the all the
		// peers in your group
		for _, peer := range self.peers {
			peer.conn.WriteMessage(ws.TextMessage, []byte(rest))
		}
	case "add": // cmd: `add <id>` where `id` is that of the connection that
		// is to be added to your group
		if other, ok := store[rest]; ok {
			self.peers = append(self.peers, &peer{uuid: other.uuid, conn: other.host})
			self.host.WriteMessage(ws.TextMessage, []byte("added to self"))
			other.host.WriteMessage(ws.TextMessage, self.banner)
		}
	case "banner": // cmd: `banner <info>` where `info` is the details to be
		// set on the banner which will be writen to the first joinee to your group
		self.banner = []byte(rest)
	}

	return false
}

func clear_self(self *group) func(code int, text string) error {
	return func(code int, text string) error {
		for _, other := range store {
			if other.uuid == self.uuid {
				continue
			}

			// TODO optimize later
			var updated_peers []*peer
			for _, peer := range other.peers {
				if peer.uuid == self.uuid {
					continue
				}

				updated_peers = append(updated_peers, peer)
			}

			other.peers = updated_peers
		}

		delete(store, self.uuid)
		return nil
	}
}
