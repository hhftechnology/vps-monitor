package containerstats

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
	"github.com/hhftechnology/vps-monitor/internal/services"
)

type statsStore interface {
	InsertContainerStat(stat models.ContainerStats) error
	PruneContainerStatsOlderThan(cutoff time.Time) error
}

// Collector periodically samples running container stats and stores them in SQLite.
type Collector struct {
	registry  *services.Registry
	store     statsStore
	interval  time.Duration
	retention time.Duration

	stopCh    chan struct{}
	wg        sync.WaitGroup
	lastPrune time.Time
}

func NewCollector(registry *services.Registry, store statsStore, interval, retention time.Duration) *Collector {
	return &Collector{
		registry:  registry,
		store:     store,
		interval:  interval,
		retention: retention,
		stopCh:    make(chan struct{}),
	}
}

func (c *Collector) Start() {
	if c.registry == nil || c.store == nil || c.interval <= 0 {
		return
	}

	c.wg.Add(1)
	go c.loop()
}

func (c *Collector) Stop() {
	select {
	case <-c.stopCh:
		return
	default:
		close(c.stopCh)
	}
	c.wg.Wait()
}

func (c *Collector) loop() {
	defer c.wg.Done()

	c.collectOnce()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.collectOnce()
		case <-c.stopCh:
			return
		}
	}
}

func (c *Collector) collectOnce() {
	dockerClient, releaseDocker := c.registry.AcquireDocker()
	if dockerClient == nil {
		releaseDocker()
		return
	}
	defer releaseDocker()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, host := range dockerClient.GetHosts() {
		stats, err := dockerClient.GetAllContainersStats(ctx, host.Name)
		if err != nil {
			log.Printf("container stats collector: failed to sample host %s: %v", host.Name, err)
			continue
		}
		for _, stat := range stats {
			if err := c.store.InsertContainerStat(stat); err != nil {
				log.Printf("container stats collector: failed to persist sample for %s on %s: %v", stat.ContainerID, stat.Host, err)
			}
		}
	}

	if c.retention > 0 && (c.lastPrune.IsZero() || time.Since(c.lastPrune) >= time.Hour) {
		if err := c.store.PruneContainerStatsOlderThan(time.Now().Add(-c.retention)); err != nil {
			log.Printf("container stats collector: failed to prune old samples: %v", err)
		} else {
			c.lastPrune = time.Now()
		}
	}
}
