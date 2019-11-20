package chat

type Message struct {
	Act string `json:"act"`
	Login string `json:"login"`
	UserID int `json:"userid"`
	Body string `json:"body"`
	MsgUnixTime int64 `json:"time"`
	To int `json:"to"`

}

func (self *Message) String() string {
	return "Act: "+ self.Act + " | User: " + self.Login + " | Body: " + self.Body
}
