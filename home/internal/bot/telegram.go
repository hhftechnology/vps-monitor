package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/config"
	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

const telegramAPIBase = "https://api.telegram.org"

type Service struct {
	mu       sync.Mutex
	registry *services.Registry
	client   *http.Client
	cfg      config.BotConfig
	running  bool
	stopCh   chan struct{}
	doneCh   chan struct{}
	offset   int64
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
		cfg: cfg,
	}
}

func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running || !isConfigured(s.cfg) {
		return
	}

	s.stopCh = make(chan struct{})
	s.doneCh = make(chan struct{})
	s.running = true

	go s.pollLoop(s.cfg, s.stopCh, s.doneCh)
}

func (s *Service) Stop() {
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
	s.mu.Unlock()

	s.Start()
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

		reply := s.handleCommand(strings.TrimSpace(update.Message.Text))
		if reply == "" {
			continue
		}

		if err := s.sendMessage(context.Background(), cfg.TelegramToken, strconv.FormatInt(update.Message.Chat.ID, 10), reply); err != nil {
			log.Printf("telegram bot send failed: %v", err)
		}
	}

	return nil
}

func (s *Service) handleCommand(text string) string {
	switch {
	case strings.HasPrefix(text, "/help"), strings.HasPrefix(text, "/start"):
		return "Available commands:\n/status - current container health with history\n/critical - latest critical alerts\n/help - command list"
	case strings.HasPrefix(text, "/critical"):
		return s.buildCriticalMessage()
	case strings.HasPrefix(text, "/status"):
		return s.buildStatusMessage()
	default:
		return "Unknown command. Use /help."
	}
}

func (s *Service) buildCriticalMessage() string {
	monitor := s.registry.Alerts()
	if monitor == nil {
		return "Alert monitoring is disabled."
	}

	alertsList := monitor.GetHistory().GetAll()
	critical := make([]models.Alert, 0, len(alertsList))
	for _, alert := range alertsList {
		if alert.Type == models.AlertCPUThreshold || alert.Type == models.AlertMemoryThreshold {
			critical = append(critical, alert)
		}
	}

	if len(critical) == 0 {
		return "No critical alerts."
	}

	sort.SliceStable(critical, func(i, j int) bool {
		return critical[i].Timestamp > critical[j].Timestamp
	})

	var lines []string
	lines = append(lines, "Critical alerts:")
	for _, alert := range critical[:min(5, len(critical))] {
		lines = append(lines, fmt.Sprintf("- %s on %s (%s)", alert.ContainerName, alert.Host, alert.Type))
	}
	return strings.Join(lines, "\n")
}

func (s *Service) buildStatusMessage() string {
	dockerClient, release := s.registry.AcquireDocker()
	defer release()
	if dockerClient == nil {
		return "Docker client unavailable."
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	containersMap, _, err := dockerClient.ListContainersAllHosts(ctx)
	if err != nil {
		return fmt.Sprintf("Failed to list containers: %v", err)
	}

	type containerLine struct {
		name string
		cpu  float64
		line string
	}

	var lines []containerLine
	total := 0
	running := 0
	history := s.registry.Alerts()
	var historyManager interface {
		Get1hAverages(string, string) (float64, float64, bool)
		Get12hAverages(string, string) (float64, float64, bool)
	}
	if history != nil {
		historyManager = history.GetStatsHistory()
	}

	for hostName, containers := range containersMap {
		for _, ctr := range containers {
			total++
			if ctr.State != "running" {
				continue
			}
			running++

			stats, err := dockerClient.GetContainerStatsOnce(ctx, hostName, ctr.ID)
			if err != nil {
				continue
			}

			name := ctr.ID[:12]
			if len(ctr.Names) > 0 {
				name = strings.TrimPrefix(ctr.Names[0], "/")
			}

			line := fmt.Sprintf("- %s@%s CPU %.1f%% MEM %.1f%%", name, hostName, stats.CPUPercent, stats.MemoryPercent)
			if historyManager != nil {
				cpu1h, mem1h, has1h := historyManager.Get1hAverages(hostName, ctr.ID)
				cpu12h, mem12h, has12h := historyManager.Get12hAverages(hostName, ctr.ID)
				if has1h || has12h {
					line += fmt.Sprintf(" | 1h %.1f/%.1f", cpu1h, mem1h)
					if has12h {
						line += fmt.Sprintf(" | 12h %.1f/%.1f", cpu12h, mem12h)
					}
				}
			}

			lines = append(lines, containerLine{name: name, cpu: stats.CPUPercent, line: line})
		}
	}

	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].cpu > lines[j].cpu
	})

	message := []string{
		fmt.Sprintf("Containers: %d total, %d running", total, running),
	}
	for _, line := range lines[:min(5, len(lines))] {
		message = append(message, line.line)
	}
	return strings.Join(message, "\n")
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
