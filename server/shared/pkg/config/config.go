// Package config provides configuration management for the Listen Stream system.
//
// It supports a two-tier configuration strategy:
//  1. Infrastructure configuration (from files/env): PostgreSQL, Redis, Server settings
//  2. Business configuration (from Consul KV): JWT secrets, API keys, SMS settings, feature flags
//
// Configuration is cached locally for 30 seconds to reduce Consul queries.
// A watcher can be used to receive notifications when configuration changes.
//
// Example usage:
//
//	// Load infrastructure config from file
//	fileLoader := config.NewFileLoader("config/local.yaml")
//	cfg, err := fileLoader.Load()
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Load business config from Consul
//	consulLoader, err := config.NewConsulLoader(&config.ConsulConfig{
//		Address:  "localhost:8500",
//		KVPrefix: "listen-stream",
//		CacheTTL: 30 * time.Second,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	businessCfg, err := consulLoader.Load(context.Background())
//	if err != nil {
//		log.Fatal(err)
//	}
//	cfg.Business = *businessCfg
//
//	// Watch for configuration changes
//	watcher := config.NewWatcher(consulLoader, 10 * time.Second)
//	watcher.OnChange(func(cfg *config.BusinessConfig) error {
//		log.Println("Configuration changed!")
//		// Update your services here
//		return nil
//	})
//	watcher.Start()
//	defer watcher.Stop()
package config

import (
	"context"
	"fmt"
	"time"
)

// Manager manages configuration from multiple sources.
type Manager struct {
	fileLoader   *FileLoader
	consulLoader *ConsulLoader
	watcher      *Watcher
	config       *Config
}

// ManagerConfig holds configuration for the manager.
type ManagerConfig struct {
	// File configuration
	ConfigFile string
	
	// Consul configuration
	ConsulAddress  string
	ConsulKVPrefix string
	ConsulToken    string
	
	// Cache settings
	CacheTTL time.Duration
	
	// Watch settings
	WatchEnabled  bool
	WatchInterval time.Duration
}

// NewManager creates a new configuration manager.
func NewManager(cfg *ManagerConfig) (*Manager, error) {
	// Load infrastructure config from file
	fileLoader := NewFileLoader(cfg.ConfigFile)
	config, err := fileLoader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load file config: %w", err)
	}
	
	// Create Consul loader if address is provided
	var consulLoader *ConsulLoader
	var watcher *Watcher
	
	if cfg.ConsulAddress != "" {
		cacheTTL := cfg.CacheTTL
		if cacheTTL == 0 {
			cacheTTL = 30 * time.Second
		}
		
		consulCfg := &ConsulConfig{
			Address:             cfg.ConsulAddress,
			Scheme:              "http",
			Token:               cfg.ConsulToken,
			KVPrefix:            cfg.ConsulKVPrefix,
			CacheTTL:            cacheTTL,
			ServiceName:         "listen-stream",
			HealthCheckInterval: 10 * time.Second,
			HealthCheckTimeout:  5 * time.Second,
		}
		
		consulLoader, err = NewConsulLoader(consulCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create consul loader: %w", err)
		}
		
		// Load business config
		businessCfg, err := consulLoader.Load(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to load consul config: %w", err)
		}
		config.Business = *businessCfg
		
		// Create watcher if enabled
		if cfg.WatchEnabled {
			watchInterval := cfg.WatchInterval
			if watchInterval == 0 {
				watchInterval = 10 * time.Second
			}
			watcher = NewWatcher(consulLoader, watchInterval)
		}
	}
	
	return &Manager{
		fileLoader:   fileLoader,
		consulLoader: consulLoader,
		watcher:      watcher,
		config:       config,
	}, nil
}

// GetConfig returns the current configuration.
func (m *Manager) GetConfig() *Config {
	return m.config
}

// GetBusiness returns the current business configuration.
func (m *Manager) GetBusiness() *BusinessConfig {
	return &m.config.Business
}

// GetInfrastructure returns the infrastructure configuration.
func (m *Manager) GetInfrastructure() *InfrastructureConfig {
	return &m.config.Infrastructure
}

// Reload reloads the configuration from sources.
func (m *Manager) Reload(ctx context.Context) error {
	if m.consulLoader == nil {
		return fmt.Errorf("consul loader not configured")
	}
	
	// Invalidate cache and reload
	m.consulLoader.InvalidateCache()
	
	businessCfg, err := m.consulLoader.Load(ctx)
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}
	
	m.config.Business = *businessCfg
	return nil
}

// OnChange registers a handler for configuration changes.
func (m *Manager) OnChange(handler ChangeHandler) error {
	if m.watcher == nil {
		return fmt.Errorf("watcher not enabled")
	}
	
	m.watcher.OnChange(handler)
	return nil
}

// StartWatching starts watching for configuration changes.
func (m *Manager) StartWatching() error {
	if m.watcher == nil {
		return fmt.Errorf("watcher not enabled")
	}
	
	m.watcher.Start()
	return nil
}

// StopWatching stops the configuration watcher.
func (m *Manager) StopWatching() {
	if m.watcher != nil {
		m.watcher.Stop()
	}
}

// Close closes the manager and releases resources.
func (m *Manager) Close() {
	m.StopWatching()
	
	if m.consulLoader != nil {
		m.consulLoader.Close()
	}
}
