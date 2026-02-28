package consul

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/xiaoxiao0301/listen-stream-v2/server/shared/pkg/logger"
)

// ServiceRegistry Consul服务注册器
type ServiceRegistry struct {
	client      *api.Client
	serviceID   string
	serviceName string
	logger      logger.Logger
}

// RegistryConfig 服务注册配置
type RegistryConfig struct {
	ConsulAddr  string        // Consul地址，如 "localhost:8500"
	ServiceName string        // 服务名称，如 "proxy-svc"
	ServiceAddr string        // 服务地址，如 "192.168.1.100:8002"
	ServiceTags []string      // 服务标签
	HealthCheck HealthCheckConfig // 健康检查配置
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	HTTP                           string        // HTTP健康检查地址，如 "http://192.168.1.100:8002/health"
	Interval                       time.Duration // 检查间隔，默认10s
	Timeout                        time.Duration // 检查超时，默认5s
	DeregisterCriticalServiceAfter time.Duration // 失败后多久注销，默认30s
}

// NewServiceRegistry 创建服务注册器
func NewServiceRegistry(cfg RegistryConfig, log logger.Logger) (*ServiceRegistry, error) {
	// 创建Consul客户端
	consulConfig := api.DefaultConfig()
	consulConfig.Address = cfg.ConsulAddr

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	// 生成服务ID（服务名+地址，确保唯一）
	serviceID := fmt.Sprintf("%s-%s", cfg.ServiceName, cfg.ServiceAddr)

	// 解析服务地址和端口
	host, portStr, err := net.SplitHostPort(cfg.ServiceAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid service address %s: %w", cfg.ServiceAddr, err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid port %s: %w", portStr, err)
	}

	// 构建服务注册信息
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    cfg.ServiceName,
		Address: host,
		Port:    port,
		Tags:    cfg.ServiceTags,
		Check: &api.AgentServiceCheck{
			HTTP:                           cfg.HealthCheck.HTTP,
			Interval:                       cfg.HealthCheck.Interval.String(),
			Timeout:                        cfg.HealthCheck.Timeout.String(),
			DeregisterCriticalServiceAfter: cfg.HealthCheck.DeregisterCriticalServiceAfter.String(),
		},
	}

	// 注册服务
	if err := client.Agent().ServiceRegister(registration); err != nil {
		return nil, fmt.Errorf("failed to register service: %w", err)
	}

	log.Info("Service registered to Consul",
		logger.String("service_id", serviceID),
		logger.String("service_name", cfg.ServiceName),
		logger.String("address", cfg.ServiceAddr),
	)

	return &ServiceRegistry{
		client:      client,
		serviceID:   serviceID,
		serviceName: cfg.ServiceName,
		logger:      log,
	}, nil
}

// Deregister 注销服务
func (r *ServiceRegistry) Deregister(ctx context.Context) error {
	if err := r.client.Agent().ServiceDeregister(r.serviceID); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	r.logger.Info("Service deregistered from Consul",
		logger.String("service_id", r.serviceID),
		logger.String("service_name", r.serviceName),
	)

	return nil
}

// UpdateHealthCheck 更新健康检查状态（用于自定义健康检查逻辑）
func (r *ServiceRegistry) UpdateHealthCheck(status string, output string) error {
	checkID := "service:" + r.serviceID

	var err error
	switch status {
	case "passing":
		err = r.client.Agent().UpdateTTL(checkID, output, api.HealthPassing)
	case "warning":
		err = r.client.Agent().UpdateTTL(checkID, output, api.HealthWarning)
	case "critical":
		err = r.client.Agent().UpdateTTL(checkID, output, api.HealthCritical)
	default:
		return fmt.Errorf("invalid health check status: %s", status)
	}

	if err != nil {
		return fmt.Errorf("failed to update health check: %w", err)
	}

	return nil
}
