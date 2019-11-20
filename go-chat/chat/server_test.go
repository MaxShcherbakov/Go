package chat

import(
	//"fmt"
  //"log"
  "strings"
  "testing"
  "net/http"
  "net/http/httptest"

  "github.com/gorilla/websocket"
)

func TestCheckClient(t *testing.T) {
  // t.Skip()
  t.Parallel()
  server := NewServer("/test")
  result := false
  if(server.CheckClient("123") == true || server.CheckClient("123") == false) {
    result = true
  }
  if(result == false) {
  	t.Error("Check Client return error")
  }
}

var upgrader = websocket.Upgrader{}

func echo(w http.ResponseWriter, r *http.Request) {
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer c.Close()
    for {
        mt, message, err := c.ReadMessage()
        if err != nil {
            break
        }
        err = c.WriteMessage(mt, message)
        if err != nil {
            break
        }
    }
}

func TestListen(t *testing.T) {
  // t.Skip()
  t.Parallel()

  s := httptest.NewServer(http.HandlerFunc(echo))
  defer s.Close()

  server := NewServer("/test")
  go server.Listen()

  u := "ws" + strings.TrimPrefix(s.URL, "http") + "/test"

  ws, _, err := websocket.DefaultDialer.Dial(u, nil)
  if err != nil {
      t.Fatalf("%v", err)
  }
  defer ws.Close()

  client := NewClient(ws, server)
  server.Add(client)

  if(len(server.clients) == 0) {
    t.Error("Clients count error: clients count", len(server.clients))
  }

  /*var msg Message
  msg.Act = "login"
  msg.Login = "Max"
  ws.WriteJSON(&msg)

  testClient := server.GetClient("Max")
  if(testClient.name != "Max") {
    t.Error("Client login error: login",len(testClient.name))
  }*/

  server.Del(client)

  if(len(server.clients) != 0) {
    t.Error("Clients count error: clients count",len(server.clients))
  }

  server.Done()
}
