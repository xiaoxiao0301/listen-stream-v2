# Consul Server Configuration
# 3-node cluster configuration for Listen Stream

datacenter = "dc1"
data_dir   = "/consul/data"
log_level  = "INFO"
node_name  = "consul-server-1"

# Server mode
server           = true
bootstrap_expect = 3

# Bind to all interfaces for Docker networking
bind_addr    = "0.0.0.0"
client_addr  = "0.0.0.0"
advertise_addr = "{{ GetInterfaceIP \"eth0\" }}"

# Enable UI
ui_config {
  enabled = true
}

# Ports
ports {
  http  = 8500
  https = -1
  grpc  = 8502
  dns   = 8600
}

# Performance tuning
performance {
  raft_multiplier = 1
}

# Connect (service mesh) - disabled for simplicity
connect {
  enabled = false
}

# ACL - disabled for development
acl {
  enabled                  = false
  default_policy           = "allow"
  enable_token_persistence = false
}

# DNS configuration
recursors = ["8.8.8.8", "8.8.4.4"]

# Telemetry
telemetry {
  prometheus_retention_time = "60s"
  disable_hostname          = false
}

# Snapshot configuration
snapshot_agent {}
