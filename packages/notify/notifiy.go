package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func SendDiscordNotification(webhookURL, content string) error {
	message := map[string]string{"content": content}
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(messageBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to send notification: %s", resp.Status)
	}

	return nil
}
