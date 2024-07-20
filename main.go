package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

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
	peers  map[string]*peer
	banner []byte
}

var store_mutex = sync.Mutex{}
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
		self := &group{uuid: id, host: conn, peers: map[string]*peer{}}

		store_mutex.Lock()
		store[id] = self
		store_mutex.Unlock()

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
			if _, found := other.peers[self.uuid]; !found {
				other.peers[self.uuid] = &peer{uuid: self.uuid, conn: self.host}
				self.host.WriteMessage(ws.TextMessage, other.banner)
				other.host.WriteMessage(ws.TextMessage, []byte(fmt.Sprintf("joined %s", self.uuid)))
			} else {
				self.host.WriteMessage(ws.TextMessage, []byte("already a member"))
			}
		} else {
			self.host.WriteMessage(ws.TextMessage, []byte("no group found"))
		}
	case "publish": // cmd: `publish <msg>` where `msg` is sent to the all the
		// peers in your group
		full_msg := []byte(self.uuid + " " + rest)
		for _, peer := range self.peers {
			peer.conn.WriteMessage(ws.TextMessage, full_msg)
		}
	case "add": // cmd: `add <id>` where `id` is that of the connection that
		// is to be added to your group
		if other, ok := store[rest]; ok {
			if _, found := self.peers[other.uuid]; !found {
				self.peers[other.uuid] = &peer{uuid: other.uuid, conn: other.host}
				self.host.WriteMessage(ws.TextMessage, []byte("added to self"))
				other.host.WriteMessage(ws.TextMessage, self.banner)
			} else {
				self.host.WriteMessage(ws.TextMessage, []byte("already added"))
			}
		}
	case "banner": // cmd: `banner <info>` where `info` is the details to be
		// set on the banner which will be writen to the first joinee to your group
		self.banner = []byte(rest)
	case "write":
		id, msg, _ := strings.Cut(rest, " ")
		full_msg := []byte(self.uuid + " " + msg)
		for _, peer := range self.peers {
			if peer.uuid == id {
				peer.conn.WriteMessage(ws.TextMessage, full_msg)
			}
		}
	}

	return false
}

func clear_self(self *group) func(code int, text string) error {
	return func(code int, text string) error {
		store_mutex.Lock()
		defer store_mutex.Unlock()

		for _, other := range store {
			if other.uuid == self.uuid {
				continue
			}

			delete(other.peers, self.uuid)
		}

		delete(store, self.uuid)
		return nil
	}
}
