package scanner

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

// AutoScanner watches for Docker image pull events and triggers scans automatically.
type AutoScanner struct {
	registry    *services.Registry
	scanner     *ScannerService
	db          *ScanDB
	enabled     atomic.Bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
	pollInterval time.Duration

	mu              sync.RWMutex
	lastPollAt      int64
	eventsConnected map[string]bool
}

// NewAutoScanner creates a new auto-scanner.
func NewAutoScanner(registry *services.Registry, scannerSvc *ScannerService, db *ScanDB) *AutoScanner {
	a := &AutoScanner{
		registry:        registry,
		scanner:         scannerSvc,
		db:              db,
		stopCh:          make(chan struct{}),
		pollInterval:    15 * time.Minute,
		eventsConnected: make(map[string]bool),
	}
	return a
}

// Start begins event listening and polling.
func (a *AutoScanner) Start() {
	if a.enabled.Load() {
		return
	}
	a.stopCh = make(chan struct{})
	a.enabled.Store(true)
	log.Println("Auto-scanner starting...")

	// Start event listeners for each Docker host
	dockerClient, release := a.registry.AcquireDocker()
	if dockerClient == nil {
		release()
		log.Println("Auto-scanner: Docker client unavailable, will rely on polling only")
	} else {
		for _, host := range dockerClient.GetHosts() {
			a.wg.Add(1)
			go a.listenEvents(host.Name)
		}
		release()
	}

	// Start polling fallback
	a.wg.Add(1)
	go a.pollLoop()
}

// Stop stops the auto-scanner gracefully.
func (a *AutoScanner) Stop() {
	if !a.enabled.Load() {
		return
	}
	a.enabled.Store(false)
	close(a.stopCh)
	a.wg.Wait()
	log.Println("Auto-scanner stopped")
}

// SetEnabled dynamically enables or disables auto-scanning.
func (a *AutoScanner) SetEnabled(enabled bool) {
	a.enabled.Store(enabled)
}

// IsEnabled returns whether auto-scanning is enabled.
func (a *AutoScanner) IsEnabled() bool {
	return a.enabled.Load()
}

// Status returns the current auto-scanner status.
func (a *AutoScanner) Status() map[string]interface{} {
	a.mu.RLock()
	defer a.mu.RUnlock()

	connectedCopy := make(map[string]bool, len(a.eventsConnected))
	for k, v := range a.eventsConnected {
		connectedCopy[k] = v
	}

	return map[string]interface{}{
		"enabled":         a.enabled.Load(),
		"lastPollAt":      a.lastPollAt,
		"eventsConnected": connectedCopy,
	}
}

// SetPollInterval updates the polling interval.
func (a *AutoScanner) SetPollInterval(minutes int) {
	if minutes < 1 {
		minutes = 15
	}
	a.pollInterval = time.Duration(minutes) * time.Minute
}

func (a *AutoScanner) listenEvents(hostName string) {
	defer a.wg.Done()

	backoff := time.Second
	maxBackoff := 5 * time.Minute

	for {
		select {
		case <-a.stopCh:
			return
		default:
		}

		if !a.enabled.Load() {
			time.Sleep(5 * time.Second)
			continue
		}

		a.mu.Lock()
		a.eventsConnected[hostName] = false
		a.mu.Unlock()

		dockerClient, release := a.registry.AcquireDocker()
		if dockerClient == nil {
			release()
			select {
			case <-a.stopCh:
				return
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
				continue
			}
		}

		apiClient, err := dockerClient.GetClient(hostName)
		if err != nil {
			release()
			log.Printf("Auto-scanner: failed to get Docker client for %s: %v", hostName, err)
			select {
			case <-a.stopCh:
				return
			case <-time.After(backoff):
				backoff = min(backoff*2, maxBackoff)
				continue
			}
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Watch for stop signal
		go func() {
			<-a.stopCh
			cancel()
		}()

		filter := filters.NewArgs()
		filter.Add("type", string(events.ImageEventType))
		filter.Add("event", "pull")

		eventCh, errCh := apiClient.Events(ctx, events.ListOptions{Filters: filter})
		release()

		a.mu.Lock()
		a.eventsConnected[hostName] = true
		a.mu.Unlock()
		backoff = time.Second // reset backoff on successful connection

		log.Printf("Auto-scanner: listening for image pull events on %s", hostName)

	eventLoop:
		for {
			select {
			case <-a.stopCh:
				cancel()
				return
			case event, ok := <-eventCh:
				if !ok {
					cancel()
					break eventLoop
				}
				if !a.enabled.Load() {
					continue
				}
				a.handlePullEvent(hostName, event)
			case err, ok := <-errCh:
				if ok && err != nil {
					log.Printf("Auto-scanner: event stream error on %s: %v", hostName, err)
				}
				cancel()
				break eventLoop
			}
		}

		a.mu.Lock()
		a.eventsConnected[hostName] = false
		a.mu.Unlock()

		log.Printf("Auto-scanner: event stream disconnected for %s, reconnecting in %v", hostName, backoff)
		select {
		case <-a.stopCh:
			return
		case <-time.After(backoff):
			backoff = min(backoff*2, maxBackoff)
		}
	}
}

func (a *AutoScanner) handlePullEvent(hostName string, event events.Message) {
	imageRef := event.Actor.Attributes["name"]
	if imageRef == "" {
		return
	}

	// Rate limit: skip if scanned within last 5 minutes
	state, err := a.db.GetImageScanState(hostName, imageRef)
	if err == nil && state != nil {
		if time.Now().Unix()-state.LastScanAt < 300 {
			return
		}
	}

	// Check if a scan is already running for this image
	for _, job := range a.scanner.GetJobs() {
		if job.ImageRef == imageRef && job.Host == hostName {
			switch job.Status {
			case "pending", "pulling_scanner", "scanning":
				return // already scanning
			}
		}
	}

	log.Printf("Auto-scanner: new image pull detected on %s: %s, triggering scan", hostName, imageRef)
	if _, err := a.scanner.StartScan(imageRef, hostName, ""); err != nil {
		log.Printf("Auto-scanner: failed to start scan for %s on %s: %v", imageRef, hostName, err)
	}
}

func (a *AutoScanner) pollLoop() {
	defer a.wg.Done()

	// Initial delay to let things settle
	select {
	case <-a.stopCh:
		return
	case <-time.After(30 * time.Second):
	}

	for {
		if a.enabled.Load() {
			a.pollForChanges()
		}

		select {
		case <-a.stopCh:
			return
		case <-time.After(a.pollInterval):
		}
	}
}

func (a *AutoScanner) pollForChanges() {
	a.mu.Lock()
	a.lastPollAt = time.Now().Unix()
	a.mu.Unlock()

	dockerClient, release := a.registry.AcquireDocker()
	if dockerClient == nil {
		release()
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	imagesByHost, _, err := dockerClient.ListImagesAllHosts(ctx)
	release()
	if err != nil {
		log.Printf("Auto-scanner poll: failed to list images: %v", err)
		return
	}

	for hostName, images := range imagesByHost {
		for _, img := range images {
			if len(img.RepoTags) == 0 {
				continue
			}
			imageRef := img.RepoTags[0]
			imageID := img.ID

			state, err := a.db.GetImageScanState(hostName, imageRef)
			if err != nil {
				continue
			}

			needsScan := false
			if state == nil {
				needsScan = true // never scanned
			} else if state.ImageID != "" && state.ImageID != imageID {
				needsScan = true // image changed
			}

			if !needsScan {
				continue
			}

			// Rate limit
			if state != nil && time.Now().Unix()-state.LastScanAt < 300 {
				continue
			}

			log.Printf("Auto-scanner poll: image changed on %s: %s, triggering scan", hostName, imageRef)
			if _, err := a.scanner.StartScan(imageRef, hostName, ""); err != nil {
				log.Printf("Auto-scanner poll: failed to start scan for %s on %s: %v", imageRef, hostName, err)
			}

			// Update state with new image ID
			a.db.UpsertImageScanState(hostName, imageRef, imageID, time.Now().Unix(), "")
		}
	}
}
