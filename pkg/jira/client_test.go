package jira

import (
	"encoding/base64"
	"testing"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://jira.example.com/"
	username := "user@example.com"
	token := "my-secret-token"

	client := NewClient(baseURL, username, token)

	if client.baseURL != "https://jira.example.com" {
		t.Errorf("expected baseURL 'https://jira.example.com', got '%s'", client.baseURL)
	}

	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+token))
	if client.authHeader != expectedAuth {
		t.Errorf("expected authHeader '%s', got '%s'", expectedAuth, client.authHeader)
	}
}
