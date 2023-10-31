package slack

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

type Message struct {
	Name       string `json:"name"`
	Conclusion string `json:"conclusion"`
	CheckUrl   string `json:"check_url"`
	CommitUrl  string `json:"commit_url"`
}

func PostMessage(webhookURL string, message Message) {
	b, _ := json.Marshal(message)
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
