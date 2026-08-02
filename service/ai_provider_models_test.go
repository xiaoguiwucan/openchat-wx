package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestFetchModels(t *testing.T) {
	t.Setenv("OPENCHAT_DOCKER", "false")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected authorization header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"vision-model"},{"id":"chat-model"},{"id":"chat-model"},{"id":""}]}`))
	}))
	defer server.Close()

	service := &AIProviderService{ctx: context.Background()}
	result, err := service.FetchModels(AIProviderModelsInput{BaseURL: server.URL + "/v1", APIKey: "test-key"})
	if err != nil {
		t.Fatalf("FetchModels returned error: %v", err)
	}
	want := []string{"chat-model", "vision-model"}
	if !reflect.DeepEqual(result.Models, want) || result.Count != len(want) {
		t.Fatalf("unexpected models: %#v", result)
	}
}

func TestFetchModelsRejectsEmptyCredentials(t *testing.T) {
	service := &AIProviderService{ctx: context.Background()}
	if _, err := service.FetchModels(AIProviderModelsInput{}); err == nil {
		t.Fatal("expected empty credentials to fail")
	}
}
