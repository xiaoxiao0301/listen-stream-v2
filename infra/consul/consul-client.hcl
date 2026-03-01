# Consul Client Configuration
# Used by application services when running in Docker

datacenter = "dc1"
data_dir   = "/consul/data"
log_level  = "WARN"

# Client mode (not server)
server = false

# Retry join server nodes
retry_join = ["consul-server-1", "consul-server-2", "consul-server-3"]

# Bind
bind_addr   = "0.0.0.0"
client_addr = "0.0.0.0"

# Ports
ports {
  http = 8500
  dns  = 8600
}

# DNS recursors
recursors = ["8.8.8.8"]
