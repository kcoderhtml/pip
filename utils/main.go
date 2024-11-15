package utils

import (
	"encoding/json"
	"net/http"
	"strings"
)

func GetLang(text string) (string, error) {
	resp, err := http.Post("https://guesslang.dunkirk.sh/detect", "text/plain", strings.NewReader(text))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if lang, ok := result["language"].(string); ok {
		return lang, nil
	}

	return "", nil
}
