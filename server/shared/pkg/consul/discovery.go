package consul

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// ServiceDiscovery Consul服务发现
type ServiceDiscovery struct {
	client       *api.Client
	logger       logger.Logger
	cache        map[string][]*api.ServiceEntry
	cacheMu      sync.RWMutex
	cacheExpiry  time.Duration
	lastUpdate   map[string]time.Time
	updateMu     sync.RWMutex
}

// NewServiceDiscovery 创建服务发现客户端
func NewServiceDiscovery(consulAddr string, cacheExpiry time.Duration, log logger.Logger) (*ServiceDiscovery, error) {
	consulConfig := api.DefaultConfig()
	consulConfig.Address = consulAddr
	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	return &ServiceDiscovery{
		client:      client,
		logger:      log,
		cache:       make(map[string][]*api.ServiceEntry),
		cacheExpiry: cacheExpiry,
		lastUpdate:  make(map[string]time.Time),
	}, nil
}

// GetServiceAddress 获取服务地址（带负载均衡）
// 返回格式: "host:port"
func (d *ServiceDiscovery) GetServiceAddress(serviceName string) (string, error) {
	services, err := d.getServicesFromCache(serviceName)
	if err != nil {
		return "", err
	}

	if len(services) == 0 {
		return "", fmt.Errorf("no healthy service found for %s", serviceName)
	}

	// 随机负载均衡
	service := services[rand.Intn(len(services))]
	address := fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)
	return address, nil
}

// GetAllServiceAddresses 获取所有健康服务实例的地址
func (d *ServiceDiscovery) GetAllServiceAddresses(serviceName string) ([]string, error) {
	services, err := d.getServicesFromCache(serviceName)
	if err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no healthy service found for %s", serviceName)
	}

	addresses := make([]string, len(services))
	for i, service := range services {
		addresses[i] = fmt.Sprintf("%s:%d", service.Service.Address, service.Service.Port)
	}
	return addresses, nil
}

// getServicesFromCache 从缓存获取服务，如果过期则刷新
func (d *ServiceDiscovery) getServicesFromCache(serviceName string) ([]*api.ServiceEntry, error) {
	d.updateMu.RLock()
	lastUpdate, exists := d.lastUpdate[serviceName]
	d.updateMu.RUnlock()
	needUpdate := !exists || time.Since(lastUpdate) > d.cacheExpiry
	if needUpdate {
		if err := d.refreshServiceCache(serviceName); err != nil {
			d.cacheMu.RLock()
			services, ok := d.cache[serviceName]
			d.cacheMu.RUnlock()
			if ok && len(services) > 0 {
				d.logger.Warn("Failed to refresh service cache, using stale cache",
					logger.String("service", serviceName),
					logger.String("error", err.Error()),
				)
				return services, nil
			}
			return nil, fmt.Errorf("failed to discover service %s: %w", serviceName, err)
		}
	}

	d.cacheMu.RLock()
	services := d.cache[serviceName]
	d.cacheMu.RUnlock()
	return services, nil
}

// refreshServiceCache 刷新服务缓存
func (d *ServiceDiscovery) refreshServiceCache(serviceName string) error {
	services, _, err := d.client.Health().Service(serviceName, "", true, nil)
	if err != nil {
		return fmt.Errorf("failed to query service: %w", err)
	}

	d.cacheMu.Lock()
	d.cache[serviceName] = services
	d.cacheMu.Unlock()

	d.updateMu.Lock()
	d.lastUpdate[serviceName] = time.Now()
	d.updateMu.Unlock()

	d.logger.Debug("Service cache refreshed",
		logger.String("service", serviceName),
		logger.Int("count", len(services)),
	)
	return nil
}

// Watch 监听服务变化（阻塞调用，应在goroutine中执行）
func (d *ServiceDiscovery) Watch(serviceName string, interval time.Duration, onChange func(addresses []string)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	var lastAddresses []string
	for range ticker.C {
		addresses, err := d.GetAllServiceAddresses(serviceName)
		if err != nil {
			d.logger.Error("Failed to watch service",
				logger.String("service", serviceName),
				logger.String("error", err.Error()),
			)
			continue
		}

		if !stringSliceEqual(lastAddresses, addresses) {
			d.logger.Info("Service addresses changed",
				logger.String("service", serviceName),
				logger.Any("old", lastAddresses),
				logger.Any("new", addresses),
			)
			lastAddresses = addresses
			onChange(addresses)
		}
	}
}

// stringSliceEqual 比较两个字符串切片是否相等
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// InvalidateCache 手动清除缓存
func (d *ServiceDiscovery) InvalidateCache(serviceName string) {
	d.cacheMu.Lock()
	delete(d.cache, serviceName)
	d.cacheMu.Unlock()

	d.updateMu.Lock()
	delete(d.lastUpdate, serviceName)
	d.updateMu.Unlock()

	d.logger.Debug("Service cache invalidated", logger.String("service", serviceName))
}
