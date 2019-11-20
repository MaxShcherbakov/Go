package chat

import (
	"fmt"
	"time"
	//"io"
	"log"
	"math/rand"
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/go-redis/redis"
)

const channelBufSize = 100

var maxId int = 0

// Chat client.
type Client struct {
	id     int
	ws     *websocket.Conn
	name   string
	server *Server
	ch     chan *Message
	doneCh chan bool
	color  string
	online bool
	rdb *redis.Client
}

type simpleClient struct {
	Id     int `json:"id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
}

type RGBColor struct {
	Red   int
	Green int
	Blue  int
}

func getHex(num int) string {
	hex := fmt.Sprintf("%x", num)
	if len(hex) == 1 {
		hex = "0" + hex
	}
	return hex
}

func GetRandomHEXColor() string {
	rand.Seed(time.Now().UnixNano())
	Red := rand.Intn(255)
	Green := rand.Intn(255)
	blue := rand.Intn(255)
	color := RGBColor{Red, Green, blue}
	hex := "#" + getHex(color.Red) + getHex(color.Green) + getHex(color.Blue)
	return hex
}

// Create new chat client.
func NewClient(ws *websocket.Conn, server *Server) *Client {

	if ws == nil {
		panic("ws cannot be nil")
	}

	if server == nil {
		panic("server cannot be nil")
	}

	maxId++
	ch := make(chan *Message, channelBufSize)
	doneCh := make(chan bool)
	color := GetRandomHEXColor()
	rdb := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	ws.SetCloseHandler(func(code int, text string) error{
		if err := ws.Close(); err != nil {
			server.Err(err)
		}
		message := websocket.FormatCloseMessage(code, "")
		ws.WriteControl(websocket.CloseMessage, message, time.Now().Add(time.Second))
		doneCh <- true
		return nil
	})

	return &Client{maxId, ws, "", server, ch, doneCh, color, false, rdb}
}

func (c *Client) Conn() *websocket.Conn {
	return c.ws
}

func (c *Client) Write(msg *Message) {
	select {
		case c.ch <- msg:
		default:
			c.server.Del(c)
			err := fmt.Errorf("client %d is disconnected.", c.id)
			c.server.Err(err)
	}
}

func (c *Client) Done() {
	c.doneCh <- true
}

// Listen Write and Read request via chanel
func (c *Client) Listen() {
	go c.listenWrite()
	c.listenRead()
}

func (c *Client) Login(login string) {
	c.name = login
}

func (c *Client) isLogin() bool {
	result := false
	if(c.name != "")	{
		result = true
	}
	return result
}

func (c *Client) setOnline(online bool) {
	c.online = online
}

func (c *Client) isOnline() bool {
	return c.online
}

// Listen write request via chanel
func (c *Client) listenWrite() {
	log.Println("Listening write to client")
	for {
		select {

		// send message to the client
		case msg := <-c.ch:
			log.Println("Send:", msg)
			c.ws.WriteJSON(msg)

		// receive done request
		case <-c.doneCh:
			c.doneCh <- true // for listenRead method
			return
		}
	}
}

// Listen read request via chanel
func (c *Client) listenRead() {
	log.Println("Listening read from client")
	for {
		select {

		// receive done request
		case <-c.doneCh:
			c.setOnline(false)
			c.server.sendClientsList()
			c.server.sendServerMsgToAll(c.name+" is disconnected.")
			log.Println(c.name+" is disconnected. (2)")
			if(c.isLogin() == false){
				c.server.Del(c)
			}
			c.doneCh <- true // for listenWrite method
			return

		// read data from websocket connection
		default:
			var msg Message
			err := c.ws.ReadJSON(&msg)
			msg.MsgUnixTime = time.Now().Unix()
			if err != nil {
				c.server.Err(err)
			} else {
				switch msg.Act {
					case "login":
						log.Println("Login try by", msg.Login, "=", c.server.CheckClient(msg.Login))
						var result Message
						result.Act = "loginResult"
						if(c.server.CheckClient(msg.Login) == false) {
							c.Login(msg.Login)
							result.Body = "True"
							result.Login = c.name
							result.UserID = c.id
							c.Write(&result)
							if(msg.UserID != 0) {
								if(c.server.GetClientById(msg.UserID) == nil) {
									c.id = msg.UserID
								}
							}
							c.setOnline(true)
							c.server.sendClientsList()
							c.server.sendPastMessages(c)
							c.server.sendServerMsgToAll(c.name+" is connected.")
						} else {
							existClient := c.server.GetClient(msg.Login)
							log.Println("Is client", msg.Login, "online =", existClient.online)
							if(existClient.isOnline() == false) {
								c.id = existClient.id
								c.name = existClient.name
								c.color = existClient.color
								c.setOnline(true)
								c.server.sendClientsList()
								c.server.sendPastMessages(c)
								c.server.sendServerMsgToAll(c.name+" is reconnected.")
								result.Body = "True"
								result.Login = c.name
								result.UserID = c.id
								c.Write(&result)
								c.server.Del(existClient)
							} else {
								result.Body = "User already exists"
								c.Write(&result)
							}
						}

					case "message":
						var result Message
						result.Act = "msgResult"
						result.Login = c.name
						result.UserID = c.id
						if(c.isLogin() == true) {
							result.Body = "True"
							msgDB, _ := json.Marshal(msg)
							err = c.rdb.Publish("testChannel", string(msgDB)).Err()
							if err != nil {
								c.server.Err(err)
							}
							err = c.rdb.RPush("messages", string(msgDB)).Err()
							if err != nil {
								c.server.Err(err)
							}
						} else {
							result.Body = "Authorization error"
						}
						c.Write(&result)
				}
			}
		}
	}
}
