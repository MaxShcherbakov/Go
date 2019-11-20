package chat

import (
	"log"
	"net/http"
	"encoding/json"
	"os"
	"html"
	"time"

	"github.com/gorilla/websocket"
	"github.com/go-redis/redis"
)

type Config struct {
	SendPastMessages bool `json:"SendPastMessages"`
	CountPastMessages int64 `json:"CountPastMessages"`
	SendServerMessages bool `json:"SendServerMessages"`
}

func LoadConfig(file string) (Config, error) {
	var config Config
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		return config, err
	}
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	return config, err
}

// Chat server.
type Server struct {
	pattern   string
	messages  []*Message
	clients   map[int]*Client
	addCh     chan *Client
	delCh     chan *Client
	sendAllCh chan *Message
	doneCh    chan bool
	errCh     chan error
	rdb 	  *redis.Client
	config	  Config

}

// Create new chat server.
func NewServer(pattern string) *Server {
	messages := []*Message{}
	clients := make(map[int]*Client)
	addCh := make(chan *Client)
	delCh := make(chan *Client)
	sendAllCh := make(chan *Message)
	doneCh := make(chan bool)
	errCh := make(chan error)
	rdb := redis.NewClient(&redis.Options{
		Addr: ":6379",
	})
	config, _ := LoadConfig("config.json")
	log.Println("config", config)

	return &Server{
		pattern,
		messages,
		clients,
		addCh,
		delCh,
		sendAllCh,
		doneCh,
		errCh,
		rdb,
		config,
	}
}

func (s *Server) Add(c *Client) {
	s.addCh <- c
}

func (s *Server) Del(c *Client) {
	s.delCh <- c
}

func (s *Server) CheckClient(login string) bool {
	result := false
	for _, c := range s.clients {
		if c.name == login {
			result = true
		}
	}
	return result
}

func (s *Server) GetClient(login string) *Client {
	var client *Client
	for _, c := range s.clients {
		if c.name == login {
			client = c
		}
	}
	return client
}

func (s *Server) GetClientById(id int) *Client {
	var client *Client
	for _, c := range s.clients {
		if c.id == id {
			client = c
		}
	}
	return client
}

func (s *Server) SendAll(msg *Message) {
	s.sendAllCh <- msg
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
}

func (s *Server) sendPastMessages(c *Client) {
	if(s.config.SendPastMessages == true) {
		result, err := s.rdb.LRange("messages", 0-s.config.CountPastMessages, -1).Result()
		if err != nil {
			panic(err)
		}
		for _, msgDB := range result {
			var msg Message
			err := json.Unmarshal([]byte(msgDB), &msg)
			if err != nil {
				panic(err)
			}
			if(msg.To == 0 || msg.To == c.id || msg.UserID == c.id) {
				c.Write(&msg)
			}
		}
	}
}

func (s *Server) sendAll(msg *Message) {
	msg.Body = html.EscapeString(msg.Body)
	for _, c := range s.clients {
		if(c.isOnline() == true) {
			if(msg.To == 0 || msg.To == c.id || msg.UserID == c.id) {
				c.Write(msg)
			}
		}
	}
}

func (s *Server) sendClientsList() {
	clients := make([]simpleClient, 0)
	for _, cs := range s.clients {
		if(cs.isOnline() == true){
			sc := simpleClient{Id: cs.id, Name: cs.name, Color: cs.color}
			clients = append(clients, sc)
		}
	}
	var jsonData []byte
	jsonData, _ = json.Marshal(clients)
	for _, c := range s.clients {
		if (c.isLogin() == true && c.isOnline() == true) {
			var result Message
			result.Act = "clientsListUpdate"
			result.Login = c.name
			result.Body = string(jsonData)
			c.Write(&result)
		}
	}
}

func (s *Server) sendServerMsgToAll(msg string) {
	if(s.config.SendServerMessages == true) {
		var result Message
		result.Act = "serverMsg"
		result.Body = msg
		result.MsgUnixTime = time.Now().Unix()
		for _, c := range s.clients {
			if(c.isOnline() == true) {
				c.Write(&result);
			}
		}
	}
}


// Listen and serve.
// It serves client connection and broadcast request.
func (s *Server) Listen() {

	log.Println("Listening server...")

	var upgrader = websocket.Upgrader{}
	// websocket handler
	onConnected := func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.errCh <- err
			return
		}
		defer c.Close()

		client := NewClient(c, s)
		s.Add(client)
		client.Listen()

	}
	http.HandleFunc(s.pattern, onConnected)
	log.Println("Created handler")

	go func(){
		pubsub := s.rdb.Subscribe("testChannel")

		for{
			msgDB, _ := pubsub.ReceiveMessage()
			var msg Message
			err := json.Unmarshal([]byte(msgDB.Payload), &msg)
			if err != nil {
				panic(err)
			}
			go func(){
				s.sendAll(&msg)
			}()
		}
	}()

	for {
		select {

		// Add new a client
		case c := <-s.addCh:
			log.Println("Added new client")
			s.clients[c.id] = c
			log.Println("Now", len(s.clients), "clients connected.")
			s.sendClientsList()

		// del a client
		case c := <-s.delCh:
			log.Println("Delete client", c.id, c.name)
			delete(s.clients, c.id)
			log.Println("Now", len(s.clients), "clients connected.")
			s.sendClientsList()

		// broadcast message for all clients
		case msg := <-s.sendAllCh:
			log.Println("Send all:", msg)
			s.messages = append(s.messages, msg)
			s.sendAll(msg)

		case err := <-s.errCh:
			log.Println("Error:", err.Error())

		case <-s.doneCh:
			return
		}
	}
}
