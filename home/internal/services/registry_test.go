package services

import (
	"testing"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/docker"
)

func TestSwapDockerWaitsForActiveLease(t *testing.T) {
	oldClient := &docker.MultiHostClient{}
	newClient := &docker.MultiHostClient{}
	registry := NewRegistry(oldClient, nil, nil, &config.Config{}, nil)

	leased, release := registry.AcquireDocker()
	if leased != oldClient {
		t.Fatalf("expected old client lease")
	}

	done := make(chan struct{})
	go func() {
		registry.SwapDocker(newClient)
		close(done)
	}()

	observed := make(chan struct{})
	go func() {
		for {
			client, releaseObserved := registry.AcquireDocker()
			if client == newClient {
				releaseObserved()
				close(observed)
				return
			}
			releaseObserved()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-observed:
	case <-time.After(2 * time.Second):
		t.Fatalf("did not observe new client acquisition after swap")
	}

	select {
	case <-done:
		t.Fatalf("SwapDocker returned before lease was released")
	case <-time.After(75 * time.Millisecond):
	}

	release()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("SwapDocker did not complete after lease release")
	}
}
