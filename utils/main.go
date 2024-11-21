package utils

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

type LangDetector struct {
	BaseURL string
}

func NewLangDetector(baseURL string) (*LangDetector, error) {
	detector := &LangDetector{BaseURL: baseURL}

	// test the baseURL
	if err := detector.testBaseURL(); err != nil {
		return nil, err
	}

	return detector, nil
}

func (ld *LangDetector) GetLang(text string) (string, error) {
	resp, err := http.Post(ld.BaseURL+"/detect", "text/plain", strings.NewReader(text))
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

	return "unkown", nil
}

func (ld *LangDetector) testBaseURL() error {
	// run a getlang on some go code
	lang, err := ld.GetLang("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}")

	if err != nil {
		return err
	}

	if lang != "Go" {
		return errors.New("unexpected language: " + lang)
	}

	return nil
}
