package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now, similar to CORS
	},
}

type ResizeMessage struct {
	Type string `json:"type"`
	Cols uint   `json:"cols"`
	Rows uint   `json:"rows"`
}

const terminalBufferSize = 32 * 1024

func (ar *APIRouter) HandleTerminal(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	host := r.URL.Query().Get("host")

	if host == "" {
		http.Error(w, "host parameter is required", http.StatusBadRequest)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade failed: %v", err)
		return
	}
	defer ws.Close()

	ctx := r.Context()

	execID, resp, err := ar.startExecSession(ctx, host, id)
	if err != nil {
		log.Printf("terminal session init failed: %v", err)
		if writeErr := ws.WriteMessage(websocket.TextMessage, []byte("Error creating terminal session: "+err.Error())); writeErr != nil {
			log.Printf("failed to send error message to websocket: %v", writeErr)
		}
		return
	}
	defer resp.Close()

	done := make(chan struct{})

	go streamContainerOutput(resp.Reader, ws, done)
	go ar.forwardClientInput(ctx, host, execID, resp.Conn, io.NopCloser(resp.Conn), ws)

	<-done
}

func (ar *APIRouter) startExecSession(ctx context.Context, host, containerID string) (string, *types.HijackedResponse, error) {
	execID, err := ar.docker.CreateExec(ctx, host, containerID)
	if err != nil {
		return "", nil, fmt.Errorf("create exec failed: %w", err)
	}

	resp, err := ar.docker.AttachExec(ctx, host, execID)
	if err != nil {
		return "", nil, fmt.Errorf("attach exec failed: %w", err)
	}

	return execID, resp, nil
}

func streamContainerOutput(reader io.Reader, ws *websocket.Conn, done chan<- struct{}) {
	defer close(done)

	buffer := make([]byte, terminalBufferSize)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			if writeErr := ws.WriteMessage(websocket.BinaryMessage, buffer[:n]); writeErr != nil {
				log.Printf("error writing to websocket: %v", writeErr)
				return
			}
		}

		if err != nil {
			if err != io.EOF {
				log.Printf("error reading from container: %v", err)
			}
			return
		}
	}
}

func (ar *APIRouter) forwardClientInput(
	ctx context.Context,
	host,
	execID string,
	writer io.Writer,
	closer io.Closer,
	ws *websocket.Conn,
) {
	defer closer.Close()

	for {
		messageType, data, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("websocket closed unexpectedly: %v", err)
			}
			return
		}

		if messageType == websocket.TextMessage {
			var msg ResizeMessage
			if err := json.Unmarshal(data, &msg); err == nil && msg.Type == "resize" {
				if err := ar.docker.ResizeExec(ctx, host, execID, msg.Rows, msg.Cols); err != nil {
					log.Printf("failed to resize terminal: %v", err)
				}
				continue
			}
		}

		if _, err := writer.Write(data); err != nil {
			log.Printf("failed to write to container: %v", err)
			return
		}
	}
}
