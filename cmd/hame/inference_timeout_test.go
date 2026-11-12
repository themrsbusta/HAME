package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGenerateWaitsForSlowBackendAndKeepsTokenLimit(t *testing.T) {
	limit := make(chan int, 1)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			MaxTokens int `json:"max_tokens"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		limit <- body.MaxTokens
		// Simulates a long prompt before the backend returns its first token.
		time.Sleep(100 * time.Millisecond)
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"completed"}}]}`)
	}))
	defer s.Close()

	e := &Engine{client: &http.Client{Timeout: 20 * time.Millisecond}}
	started := time.Now()
	got, err := e.generate(context.Background(), &Worker{url: s.URL}, []Message{{Role: "user", Content: "long prompt"}}, 37, "")
	if err != nil {
		t.Fatalf("slow inference failed: %v", err)
	}
	if got != "completed" {
		t.Fatalf("response = %q, want completed", got)
	}
	if elapsed := time.Since(started); elapsed < 80*time.Millisecond {
		t.Fatalf("backend delay was not observed: %s", elapsed)
	}
	if n := <-limit; n != 37 {
		t.Fatalf("max_tokens = %d, want unchanged limit 37", n)
	}
}

func TestGenerateHonorsManualCancellation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-release
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"late"}}]}`)
	}))
	defer s.Close()

	e := &Engine{client: &http.Client{Timeout: 20 * time.Millisecond}}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := e.generate(ctx, &Worker{url: s.URL}, []Message{{Role: "user", Content: "cancel me"}}, 20, "")
		result <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("backend did not receive inference")
	}
	cancel()
	close(release)
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("manual cancellation did not end inference")
	}
}

func TestGenerateReportsBackendFailure(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "backend unavailable", http.StatusBadGateway)
	}))
	defer s.Close()

	e := &Engine{client: &http.Client{Timeout: 20 * time.Millisecond}}
	_, err := e.generate(context.Background(), &Worker{url: s.URL}, []Message{{Role: "user", Content: "test"}}, 20, "")
	if err == nil || !strings.Contains(err.Error(), fmt.Sprint(http.StatusBadGateway)) || !strings.Contains(err.Error(), "backend unavailable") {
		t.Fatalf("backend failure was not preserved: %v", err)
	}
}

func TestInferenceHTTPClientHasNoAbsoluteTimeout(t *testing.T) {
	e := &Engine{client: &http.Client{Timeout: 3 * time.Minute}}
	if got := e.inferenceHTTPClient().Timeout; got != 0 {
		t.Fatalf("inference timeout = %s, want no absolute timeout", got)
	}
}

func TestManualCompactionClientHasNoAbsoluteTimeout(t *testing.T) {
	if got := memoryRequestClient(true).Timeout; got != 0 {
		t.Fatalf("manual compaction timeout = %s, want no absolute timeout", got)
	}
	if got := memoryRequestClient(false).Timeout; got != 20*time.Minute {
		t.Fatalf("status request timeout = %s, want existing 20 minute control limit", got)
	}
}
