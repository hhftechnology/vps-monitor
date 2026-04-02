package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

// HandleContainerStats handles WebSocket connections for real-time container stats
func (ar *APIRouter) HandleContainerStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed for stats: %v", err)
		return
	}
	defer ws.Close()

	// Set up ping/pong keep-alive
	ws.SetReadDeadline(time.Now().Add(wsPongTimeout))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(wsPongTimeout))
		return nil
	})

	ctx := r.Context()

	// Start streaming stats from Docker
	statsCh, errCh := ar.registry.Docker().StreamContainerStats(ctx, host, id)

	// Handle WebSocket close from client
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNoStatusReceived) {
					log.Printf("stats websocket closed unexpectedly: %v", err)
				}
				return
			}
		}
	}()

	pingTicker := time.NewTicker(wsPingInterval)
	defer pingTicker.Stop()

	// Stream stats to WebSocket
	for {
		select {
		case stats, ok := <-statsCh:
			if !ok {
				return
			}
			data, err := json.Marshal(stats)
			if err != nil {
				log.Printf("failed to marshal stats: %v", err)
				continue
			}
			ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("failed to write stats to websocket: %v", err)
				return
			}

		case err := <-errCh:
			if err != nil {
				log.Printf("stats stream error: %v", err)
				errMsg := map[string]string{"error": err.Error()}
				if data, _ := json.Marshal(errMsg); data != nil {
					ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
					_ = ws.WriteMessage(websocket.TextMessage, data)
				}
			}
			return

		case <-pingTicker.C:
			ws.SetWriteDeadline(time.Now().Add(wsWriteTimeout))
			if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-done:
			return

		case <-ctx.Done():
			return
		}
	}
}

// GetContainerStatsOnce returns a single stats snapshot (HTTP endpoint)
func (ar *APIRouter) GetContainerStatsOnce(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	stats, err := ar.registry.Docker().GetContainerStatsOnce(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"stats": stats,
	})
}
