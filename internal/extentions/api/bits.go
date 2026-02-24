package api

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type api struct {
	name    string
	content string
	color   string
}

func SendBitsNotification(endpoint, name, content, nameColor string) {
	go func() {
		payload := api{
			name:    name,
			content: content,
			color:   nameColor,
		}

		body, err := json.Marshal(map[string]string{
			"name":       payload.name,
			"content":    payload.content,
			"name_color": payload.color,
		})
		if err != nil {
			return
		}

		resp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}
