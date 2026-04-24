package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hhftechnology/vps-monitor/internal/config"
)

const (
	discordAPIBase = "https://discord.com/api/v10"

	discordOpDispatch       = 0
	discordOpHeartbeat      = 1
	discordOpIdentify       = 2
	discordOpResume         = 6
	discordOpReconnect      = 7
	discordOpInvalidSession = 9
	discordOpHello          = 10
	discordOpHeartbeatACK   = 11

	discordInteractionApplicationCommand = 2
	discordResponseDeferredMessage       = 5
	discordMessageFlagEphemeral          = 64
)

type websocketDialer interface {
	Dial(urlStr string, requestHeader http.Header) (*websocket.Conn, *http.Response, error)
}

type discordGatewayResponse struct {
	URL string `json:"url"`
}

type discordGatewayPayload struct {
	Op int             `json:"op"`
	D  json.RawMessage `json:"d"`
	S  *int64          `json:"s,omitempty"`
	T  string          `json:"t,omitempty"`
}

type discordHelloPayload struct {
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type discordReadyPayload struct {
	SessionID        string `json:"session_id"`
	ResumeGatewayURL string `json:"resume_gateway_url"`
}

type discordInteraction struct {
	ID        string `json:"id"`
	Token     string `json:"token"`
	Type      int    `json:"type"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
	Data      struct {
		Name string `json:"name"`
	} `json:"data"`
}

type discordCommand struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        int    `json:"type"`
}

func (s *Service) startDiscordLocked() {
	if s.discordRunning || !isDiscordConfigured(s.cfg.Discord) {
		return
	}

	cfg := s.cfg
	s.discordStopCh = make(chan struct{})
	s.discordDoneCh = make(chan struct{})
	s.discordRunning = true

	go s.discordLoop(cfg, s.discordStopCh, s.discordDoneCh)
}

func (s *Service) stopDiscord() {
	s.mu.Lock()
	if !s.discordRunning {
		s.mu.Unlock()
		return
	}
	stopCh := s.discordStopCh
	doneCh := s.discordDoneCh
	s.discordRunning = false
	s.discordStopCh = nil
	s.discordDoneCh = nil
	s.mu.Unlock()

	close(stopCh)
	<-doneCh
}

func (s *Service) discordLoop(cfg config.BotConfig, stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)

	if err := s.registerDiscordCommands(context.Background(), cfg.Discord); err != nil {
		log.Printf("discord bot command registration failed: %v", err)
	}

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		if err := s.runDiscordConnection(cfg, stopCh); err != nil {
			log.Printf("discord bot gateway failed: %v", err)
		}

		select {
		case <-time.After(5 * time.Second):
		case <-stopCh:
			return
		}
	}
}

func (s *Service) runDiscordConnection(cfg config.BotConfig, stopCh <-chan struct{}) error {
	gatewayURL, err := s.discordGatewayURL(context.Background(), cfg.Discord.BotToken)
	if err != nil {
		return err
	}

	sessionID, resumeURL, _ := s.discordSession()
	connectURL := gatewayURL
	resuming := sessionID != "" && resumeURL != ""
	if resuming {
		connectURL = resumeURL
	}
	if !strings.Contains(connectURL, "?") {
		connectURL += "?v=10&encoding=json"
	}

	conn, _, err := s.discordDialer.Dial(connectURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	stopped := make(chan struct{})
	go func() {
		select {
		case <-stopCh:
			_ = conn.Close()
		case <-stopped:
		}
	}()
	defer close(stopped)

	var writeMu sync.Mutex
	ackMu := sync.Mutex{}
	heartbeatAcked := true
	heartbeatStop := make(chan struct{})
	defer close(heartbeatStop)

	for {
		var payload discordGatewayPayload
		if err := conn.ReadJSON(&payload); err != nil {
			select {
			case <-stopCh:
				return nil
			default:
				return err
			}
		}

		if payload.S != nil {
			s.setDiscordSeq(*payload.S)
		}

		switch payload.Op {
		case discordOpHello:
			var hello discordHelloPayload
			if err := json.Unmarshal(payload.D, &hello); err != nil {
				return err
			}
			if hello.HeartbeatInterval <= 0 {
				return fmt.Errorf("discord gateway hello missing heartbeat interval")
			}
			go s.discordHeartbeatLoop(conn, &writeMu, &ackMu, &heartbeatAcked, time.Duration(hello.HeartbeatInterval)*time.Millisecond, heartbeatStop)
			if resuming {
				if err := s.discordResume(conn, &writeMu, cfg.Discord.BotToken, sessionID); err != nil {
					return err
				}
			} else if err := s.discordIdentify(conn, &writeMu, cfg.Discord.BotToken); err != nil {
				return err
			}
		case discordOpHeartbeatACK:
			ackMu.Lock()
			heartbeatAcked = true
			ackMu.Unlock()
		case discordOpHeartbeat:
			if err := s.discordSendHeartbeat(conn, &writeMu); err != nil {
				return err
			}
		case discordOpReconnect:
			return fmt.Errorf("discord gateway requested reconnect")
		case discordOpInvalidSession:
			s.clearDiscordSession()
			return fmt.Errorf("discord gateway invalidated session")
		case discordOpDispatch:
			if payload.T == "READY" {
				var ready discordReadyPayload
				if err := json.Unmarshal(payload.D, &ready); err != nil {
					return err
				}
				s.setDiscordSession(ready.SessionID, ready.ResumeGatewayURL)
			}
			if payload.T == "INTERACTION_CREATE" {
				var interaction discordInteraction
				if err := json.Unmarshal(payload.D, &interaction); err != nil {
					log.Printf("discord interaction decode failed: %v", err)
					continue
				}
				go s.handleDiscordInteraction(context.Background(), cfg.Discord, interaction)
			}
		}
	}
}

func (s *Service) discordHeartbeatLoop(conn *websocket.Conn, writeMu, ackMu *sync.Mutex, acked *bool, interval time.Duration, stopCh <-chan struct{}) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if err := s.discordSendHeartbeat(conn, writeMu); err != nil {
		_ = conn.Close()
		return
	}
	ackMu.Lock()
	*acked = false
	ackMu.Unlock()

	for {
		select {
		case <-ticker.C:
			ackMu.Lock()
			if !*acked {
				ackMu.Unlock()
				_ = conn.Close()
				return
			}
			*acked = false
			ackMu.Unlock()

			if err := s.discordSendHeartbeat(conn, writeMu); err != nil {
				_ = conn.Close()
				return
			}
		case <-stopCh:
			return
		}
	}
}

func (s *Service) discordIdentify(conn *websocket.Conn, writeMu *sync.Mutex, token string) error {
	payload := map[string]any{
		"op": discordOpIdentify,
		"d": map[string]any{
			"token":   token,
			"intents": 0,
			"properties": map[string]string{
				"os":      "linux",
				"browser": "vps-monitor",
				"device":  "vps-monitor",
			},
		},
	}
	return discordWriteJSON(conn, writeMu, payload)
}

func (s *Service) discordResume(conn *websocket.Conn, writeMu *sync.Mutex, token, sessionID string) error {
	payload := map[string]any{
		"op": discordOpResume,
		"d": map[string]any{
			"token":      token,
			"session_id": sessionID,
			"seq":        s.discordSeq(),
		},
	}
	return discordWriteJSON(conn, writeMu, payload)
}

func (s *Service) discordSendHeartbeat(conn *websocket.Conn, writeMu *sync.Mutex) error {
	return discordWriteJSON(conn, writeMu, map[string]any{
		"op": discordOpHeartbeat,
		"d":  s.discordSeq(),
	})
}

func discordWriteJSON(conn *websocket.Conn, writeMu *sync.Mutex, payload any) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteJSON(payload)
}

func (s *Service) discordGatewayURL(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.discordAPIBase+"/gateway/bot", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bot "+token)

	res, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("discord gateway lookup returned status %d", res.StatusCode)
	}

	var payload discordGatewayResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.URL == "" {
		return "", fmt.Errorf("discord gateway lookup returned empty url")
	}
	return payload.URL, nil
}

func (s *Service) registerDiscordCommands(ctx context.Context, cfg config.DiscordBotConfig) error {
	commands := []discordCommand{
		{Name: "help", Description: "Show VPS Monitor bot commands", Type: 1},
		{Name: "status", Description: "Show current container health with history", Type: 1},
		{Name: "critical", Description: "Show latest critical alerts", Type: 1},
	}

	endpoint := fmt.Sprintf("%s/applications/%s/commands", s.discordAPIBase, cfg.ApplicationID)
	if cfg.GuildID != "" {
		endpoint = fmt.Sprintf("%s/applications/%s/guilds/%s/commands", s.discordAPIBase, cfg.ApplicationID, cfg.GuildID)
	}

	return s.doDiscordJSON(ctx, http.MethodPut, endpoint, cfg.BotToken, commands, nil)
}

func (s *Service) handleDiscordInteraction(ctx context.Context, cfg config.DiscordBotConfig, interaction discordInteraction) {
	if interaction.Type != discordInteractionApplicationCommand {
		return
	}

	if err := s.deferDiscordInteraction(ctx, cfg, interaction); err != nil {
		log.Printf("discord interaction defer failed: %v", err)
		return
	}

	reply := s.discordInteractionReply(cfg, interaction)
	if reply == "" {
		reply = "No response."
	}
	if err := s.editDiscordInteractionResponse(ctx, cfg, interaction.Token, reply); err != nil {
		log.Printf("discord interaction response failed: %v", err)
	}
}

func (s *Service) discordInteractionReply(cfg config.DiscordBotConfig, interaction discordInteraction) string {
	if cfg.GuildID != "" && interaction.GuildID != cfg.GuildID {
		return "This Discord server is not allowed."
	}
	if cfg.AllowedChannelID != "" && interaction.ChannelID != cfg.AllowedChannelID {
		return "This Discord channel is not allowed."
	}

	switch interaction.Data.Name {
	case "help", "status", "critical":
		return s.commands.handle("/" + interaction.Data.Name)
	default:
		return "Unknown command. Use /help."
	}
}

func (s *Service) deferDiscordInteraction(ctx context.Context, cfg config.DiscordBotConfig, interaction discordInteraction) error {
	endpoint := fmt.Sprintf("%s/interactions/%s/%s/callback", s.discordAPIBase, interaction.ID, interaction.Token)
	payload := map[string]any{
		"type": discordResponseDeferredMessage,
		"data": map[string]any{"flags": discordMessageFlagEphemeral},
	}
	return s.doDiscordJSON(ctx, http.MethodPost, endpoint, cfg.BotToken, payload, nil)
}

func (s *Service) editDiscordInteractionResponse(ctx context.Context, cfg config.DiscordBotConfig, token, content string) error {
	endpoint := fmt.Sprintf("%s/webhooks/%s/%s/messages/@original", s.discordAPIBase, cfg.ApplicationID, token)
	payload := map[string]any{
		"content": content,
		"flags":   discordMessageFlagEphemeral,
	}
	return s.doDiscordJSON(ctx, http.MethodPatch, endpoint, cfg.BotToken, payload, nil)
}

func (s *Service) SendDiscordTestMessage(ctx context.Context, token, channelID string) error {
	if strings.TrimSpace(token) == "" || strings.TrimSpace(channelID) == "" {
		return fmt.Errorf("discord bot token and channel id are required")
	}

	endpoint := fmt.Sprintf("%s/channels/%s/messages", s.discordAPIBase, strings.TrimSpace(channelID))
	payload := map[string]string{"content": "VPS Monitor Discord bot test successful."}
	return s.doDiscordJSON(ctx, http.MethodPost, endpoint, strings.TrimSpace(token), payload, nil)
}

func (s *Service) doDiscordJSON(ctx context.Context, method, endpoint, token string, payload, out any) error {
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("discord %s returned status %d", method, res.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

func (s *Service) discordSession() (string, string, int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.discordSessionID, s.discordResumeURL, s.discordLastSeq
}

func (s *Service) setDiscordSession(sessionID, resumeURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discordSessionID = sessionID
	s.discordResumeURL = resumeURL
}

func (s *Service) clearDiscordSession() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discordSessionID = ""
	s.discordResumeURL = ""
	s.discordLastSeq = 0
}

func (s *Service) discordSeq() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.discordLastSeq
}

func (s *Service) setDiscordSeq(seq int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discordLastSeq = seq
}

func isDiscordConfigured(cfg config.DiscordBotConfig) bool {
	return cfg.Enabled &&
		strings.TrimSpace(cfg.BotToken) != "" &&
		strings.TrimSpace(cfg.ApplicationID) != "" &&
		strings.TrimSpace(cfg.AllowedChannelID) != ""
}
