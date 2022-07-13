package utils

import (
	"bytes"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"io"
	"net/http"
)

type WebhookRequestBody struct {
	Username string `json:"username"`
	Content  string `json:"content"`
}

func SendWebhook(webhookRequestBody WebhookRequestBody) error {

	url := viper.GetString("webhook_url")

	if len(url) == 0 {
		return nil
	}

	log.Debug().Msg("sending webhook")

	body, err := json.Marshal(webhookRequestBody)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Err(err).Msg("error closing body")
		}
	}(resp.Body)

	return nil
}
