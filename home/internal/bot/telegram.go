package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

const telegramAPIBase = "https://api.telegram.org"

type Service struct {
	mu               sync.Mutex
	registry         *services.Registry
	commands         *commandHandler
	client           *http.Client
	discordAPIBase   string
	discordDialer    websocketDialer
	cfg              config.BotConfig
	running          bool
	stopCh           chan struct{}
	doneCh           chan struct{}
	offset           int64
	discordRunning   bool
	discordStopCh    chan struct{}
	discordDoneCh    chan struct{}
	discordSessionID string
	discordResumeURL string
	discordLastSeq   int64
}

type telegramUpdateResponse struct {
	OK     bool             `json:"ok"`
	Result []telegramUpdate `json:"result"`
}

type telegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *telegramMessage `json:"message"`
}

type telegramMessage struct {
	MessageID int64        `json:"message_id"`
	Text      string       `json:"text"`
	Chat      telegramChat `json:"chat"`
}

type telegramChat struct {
	ID int64 `json:"id"`
}

func NewService(registry *services.Registry, cfg config.BotConfig) *Service {
	return &Service{
		registry: registry,
		client: &http.Client{
			Timeout: 35 * time.Second,
		},
		commands:       newCommandHandler(registry),
		discordAPIBase: discordAPIBase,
		discordDialer:  websocket.DefaultDialer,
		cfg:            cfg,
	}
}

func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running && isConfigured(s.cfg) && s.cfg.Mode == config.BotModePolling {
		s.stopCh = make(chan struct{})
		s.doneCh = make(chan struct{})
		s.running = true

		go s.pollLoop(s.cfg, s.stopCh, s.doneCh)
	}

	s.startDiscordLocked()
}

func (s *Service) Stop() {
	s.stopDiscord()

	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	stopCh := s.stopCh
	doneCh := s.doneCh
	s.running = false
	s.stopCh = nil
	s.doneCh = nil
	s.mu.Unlock()

	close(stopCh)
	<-doneCh
}

func (s *Service) UpdateConfig(cfg config.BotConfig) {
	s.Stop()

	s.mu.Lock()
	s.cfg = cfg
	s.offset = 0
	s.discordLastSeq = 0
	s.discordSessionID = ""
	s.discordResumeURL = ""
	s.mu.Unlock()

	s.Start()
}

func (s *Service) RelayCommand(ctx context.Context, chatID, text string) (string, error) {
	s.mu.Lock()
	cfg := s.cfg
	s.mu.Unlock()

	if cfg.Mode != config.BotModeJWTRelay {
		return "", fmt.Errorf("bot relay mode is disabled")
	}
	if !isConfigured(cfg) {
		return "", fmt.Errorf("telegram token and allowed chat id are required")
	}

	targetChatID := strings.TrimSpace(chatID)
	if targetChatID == "" {
		targetChatID = cfg.AllowedChatID
	}
	if cfg.AllowedChatID != "" && targetChatID != cfg.AllowedChatID {
		return "", fmt.Errorf("chat id is not allowed")
	}

	reply := s.commands.handle(strings.TrimSpace(text))
	if reply == "" {
		return "", nil
	}

	if err := s.sendMessage(ctx, cfg.TelegramToken, targetChatID, reply); err != nil {
		return "", err
	}

	return reply, nil
}

func (s *Service) SendTestMessage(ctx context.Context, token, chatID string) error {
	if strings.TrimSpace(token) == "" || strings.TrimSpace(chatID) == "" {
		return fmt.Errorf("telegram token and chat id are required")
	}

	return s.sendMessage(ctx, token, chatID, "VPS Monitor bot test successful.")
}

func (s *Service) pollLoop(cfg config.BotConfig, stopCh <-chan struct{}, doneCh chan<- struct{}) {
	defer close(doneCh)

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		if err := s.pollOnce(cfg); err != nil {
			log.Printf("telegram bot poll failed: %v", err)
			select {
			case <-time.After(5 * time.Second):
			case <-stopCh:
				return
			}
		}
	}
}

func (s *Service) pollOnce(cfg config.BotConfig) error {
	params := url.Values{}
	params.Set("timeout", strconv.Itoa(int(cfg.PollInterval.Seconds())))

	s.mu.Lock()
	offset := s.offset
	s.mu.Unlock()
	if offset > 0 {
		params.Set("offset", strconv.FormatInt(offset, 10))
	}

	req, err := http.NewRequest(http.MethodGet, s.apiURL(cfg.TelegramToken, "getUpdates", params), nil)
	if err != nil {
		return err
	}

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return fmt.Errorf("telegram getUpdates returned status %d", res.StatusCode)
	}

	var payload telegramUpdateResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return err
	}

	for _, update := range payload.Result {
		s.mu.Lock()
		if update.UpdateID >= s.offset {
			s.offset = update.UpdateID + 1
		}
		s.mu.Unlock()

		if update.Message == nil {
			continue
		}

		if cfg.AllowedChatID != "" && strconv.FormatInt(update.Message.Chat.ID, 10) != cfg.AllowedChatID {
			continue
		}

		reply := s.commands.handle(strings.TrimSpace(update.Message.Text))
		if reply == "" {
			continue
		}

		if err := s.sendMessage(context.Background(), cfg.TelegramToken, strconv.FormatInt(update.Message.Chat.ID, 10), reply); err != nil {
			log.Printf("telegram bot send failed: %v", err)
		}
	}

	return nil
}

func (s *Service) sendMessage(ctx context.Context, token, chatID, text string) error {
	form := url.Values{}
	form.Set("chat_id", chatID)
	form.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiURL(token, "sendMessage", nil), strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		return fmt.Errorf("telegram sendMessage returned status %d", res.StatusCode)
	}

	return nil
}

func (s *Service) apiURL(token, method string, params url.Values) string {
	base := fmt.Sprintf("%s/bot%s/%s", telegramAPIBase, token, method)
	if params == nil || len(params) == 0 {
		return base
	}
	return base + "?" + params.Encode()
}

func isConfigured(cfg config.BotConfig) bool {
	return cfg.Enabled && strings.TrimSpace(cfg.TelegramToken) != "" && strings.TrimSpace(cfg.AllowedChatID) != ""
}
