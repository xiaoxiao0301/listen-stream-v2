package config

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
)

// Watcher watches for configuration changes in Consul KV.
type Watcher struct {
	client         *api.Client
	kvPrefix       string
	interval       time.Duration
	loader         *ConsulLoader
	lastIndex      uint64
	mu             sync.RWMutex
	changeHandlers []ChangeHandler
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// ChangeHandler is a function called when configuration changes.
type ChangeHandler func(cfg *BusinessConfig) error

// NewWatcher creates a new configuration watcher.
func NewWatcher(loader *ConsulLoader, interval time.Duration) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Watcher{
		client:         loader.client,
		kvPrefix:       loader.kvPrefix,
		interval:       interval,
		loader:         loader,
		changeHandlers: make([]ChangeHandler, 0),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// OnChange registers a handler to be called when configuration changes.
func (w *Watcher) OnChange(handler ChangeHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	w.changeHandlers = append(w.changeHandlers, handler)
}

// Start starts watching for configuration changes.
func (w *Watcher) Start() {
	w.wg.Add(1)
	go w.watch()
}

// Stop stops the watcher.
func (w *Watcher) Stop() {
	w.cancel()
	w.wg.Wait()
}

// watch watches for configuration changes.
func (w *Watcher) watch() {
	defer w.wg.Done()
	
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.checkForChanges(); err != nil {
				log.Printf("Error checking for config changes: %v", err)
			}
		}
	}
}

// checkForChanges checks if the configuration has changed.
func (w *Watcher) checkForChanges() error {
	// Use blocking query with the last known index
	queryOpts := &api.QueryOptions{
		WaitIndex: w.lastIndex,
		WaitTime:  30 * time.Second,
	}
	
	// List all keys under the prefix
	_, meta, err := w.client.KV().List(w.kvPrefix, queryOpts)
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}
	
	// Check if anything changed
	if meta.LastIndex == w.lastIndex {
		// No changes
		return nil
	}
	
	// Update last index
	w.lastIndex = meta.LastIndex
	
	// Configuration changed - reload
	log.Printf("Configuration changed (index: %d), reloading...", meta.LastIndex)
	
	// Invalidate cache
	w.loader.InvalidateCache()
	
	// Reload configuration
	cfg, err := w.loader.Load(w.ctx)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	
	// Call change handlers
	w.mu.RLock()
	handlers := append([]ChangeHandler(nil), w.changeHandlers...)
	w.mu.RUnlock()
	
	for _, handler := range handlers {
		if err := handler(cfg); err != nil {
			log.Printf("Error in change handler: %v", err)
		}
	}
	
	log.Printf("Configuration reloaded successfully")
	
	return nil
}

// WatchKey watches a specific key for changes.
// This is useful for watching a single configuration value.
func (w *Watcher) WatchKey(key string, handler func(value string) error) {
	w.wg.Add(1)
	
	go func() {
		defer w.wg.Done()
		
		var lastValue string
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				pair, _, err := w.client.KV().Get(key, nil)
				if err != nil {
					log.Printf("Error watching key %s: %v", key, err)
					continue
				}
				
				if pair == nil {
					continue
				}
				
				value := string(pair.Value)
				if value != lastValue {
					lastValue = value
					if err := handler(value); err != nil {
						log.Printf("Error in key watch handler for %s: %v", key, err)
					}
				}
			}
		}
	}()
}
