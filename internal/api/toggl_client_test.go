package api

import (
	"testing"
)

func TestNewAPIClient_ReturnsNonNil(t *testing.T) {
	client := NewAPIClient("test-token")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
}

func TestNewAPIClient_SetsAuthToken(t *testing.T) {
	client := NewAPIClient("my-token")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.AuthToken != "my-token" {
		t.Errorf("AuthToken: got %q, want %q", client.AuthToken, "my-token")
	}
}

func TestNewAPIClient_SetsBaseURL(t *testing.T) {
	client := NewAPIClient("tok")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.BaseURL == "" {
		t.Error("BaseURL should not be empty")
	}
}

func TestNewAPIClient_SetsHTTPClient(t *testing.T) {
	client := NewAPIClient("tok")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.HTTPClient == nil {
		t.Error("HTTPClient should not be nil")
	}
}

func TestNewAPIClient_SetsCache(t *testing.T) {
	client := NewAPIClient("tok")
	if client == nil {
		t.Fatal("NewAPIClient returned nil")
	}
	if client.Cache == nil {
		t.Error("Cache should not be nil")
	}
}
