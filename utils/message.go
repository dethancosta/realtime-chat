package utils

import "time"

type Message struct {
	Id int64 `json:"id"`
	Name string `json:"name"`
	Date time.Time `json:"date"`
	Room string `json:"room"`
	Content string `json:"content"`
}

func NewMessage() Message {
	message := Message{
		Date: time.Now(),
	}

	return message
}

func (m Message) String() string {
	return "--- " + m.Name + " --- " + m.Date.Format(time.Layout) + "\n" + m.Content + "\n\n"
}
