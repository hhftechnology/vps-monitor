package api

import (
	"encoding/json"
	"log"
	"net/http"

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

	ctx := r.Context()

	// Start streaming stats from Docker
	statsCh, errCh := ar.docker.StreamContainerStats(ctx, host, id)

	// Handle WebSocket close from client
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("stats websocket closed unexpectedly: %v", err)
				}
				return
			}
		}
	}()

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
			if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("failed to write stats to websocket: %v", err)
				return
			}

		case err := <-errCh:
			if err != nil {
				log.Printf("stats stream error: %v", err)
				// Send error to client
				errMsg := map[string]string{"error": err.Error()}
				if data, _ := json.Marshal(errMsg); data != nil {
					_ = ws.WriteMessage(websocket.TextMessage, data)
				}
			}
			return

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

	stats, err := ar.docker.GetContainerStatsOnce(r.Context(), host, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	WriteJsonResponse(w, http.StatusOK, map[string]any{
		"stats": stats,
	})
}
