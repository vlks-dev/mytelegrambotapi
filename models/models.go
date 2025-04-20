package models

import "time"

type Message struct {
	MessageID    int       `json:"message_id"`
	FromID       string    `json:"from_id"`
	FromUsername string    `json:"from_username"`
	Text         string    `json:"text"`
	Location     string    `json:"location"`
	Timestamp    time.Time `json:"time_stamp"`
}
