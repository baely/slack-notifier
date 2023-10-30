package slack

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

type SlackGHMessage struct {
	Message string
}

func PostMessage(webhookURL string, message string) {
	msg := SlackGHMessage{message}
	b, _ := json.Marshal(msg)
	r := bytes.NewReader(b)

	resp, err := http.Post(webhookURL, "application/json", r)
	if err != nil {
		log.Printf("Error while posting message: %s", err)
	}

	if resp.StatusCode != 200 {
		log.Printf("Error while posting message: %s", resp.Status)
	}

	log.Printf("Message posted")
}
