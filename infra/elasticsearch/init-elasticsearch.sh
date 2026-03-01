#!/bin/bash
# Initialize Elasticsearch: ILM policy, index template, initial aliases.
# Usage: ./init-elasticsearch.sh [ES_URL]
# Log retention: 30 days hot → delete

set -e

ES_URL="${1:-http://localhost:9200}"

echo "==> Initializing Elasticsearch at ${ES_URL}"

# Wait for Elasticsearch
MAX_TRIES=30
TRIES=0
until curl -sf "${ES_URL}/_cluster/health" > /dev/null 2>&1; do
    TRIES=$((TRIES + 1))
    if [ $TRIES -ge $MAX_TRIES ]; then
        echo "ERROR: Elasticsearch not ready after ${MAX_TRIES} attempts"
        exit 1
    fi
    echo "Waiting for Elasticsearch... (attempt ${TRIES}/${MAX_TRIES})"
    sleep 3
done

echo "==> Elasticsearch is ready"

# ── 1. ILM Policy: 30-day retention ──────────────────────────────────────────
echo "==> Creating ILM policy (30-day retention)..."
curl -sf -X PUT "${ES_URL}/_ilm/policy/listen-stream-30d-policy" \
  -H "Content-Type: application/json" -d '{
  "policy": {
    "phases": {
      "hot": {
        "min_age": "0ms",
        "actions": {
          "rollover": {
            "max_primary_shard_size": "5gb",
            "max_age": "1d"
          },
          "set_priority": { "priority": 100 }
        }
      },
      "warm": {
        "min_age": "7d",
        "actions": {
          "readonly": {},
          "shrink": { "number_of_shards": 1 },
          "forcemerge": { "max_num_segments": 1 },
          "set_priority": { "priority": 50 }
        }
      },
      "cold": {
        "min_age": "20d",
        "actions": {
          "set_priority": { "priority": 0 },
          "readonly": {}
        }
      },
      "delete": {
        "min_age": "30d",
        "actions": {
          "delete": {
            "delete_searchable_snapshot": true
          }
        }
      }
    }
  }
}' > /dev/null
echo "  ✓ ILM policy created"

# ── 2. Index Template for service logs ───────────────────────────────────────
echo "==> Creating index template..."
curl -sf -X PUT "${ES_URL}/_index_template/listen-stream-logs" \
  -H "Content-Type: application/json" -d '{
  "index_patterns": ["listen-stream-logs-*"],
  "priority": 200,
  "template": {
    "settings": {
      "number_of_shards":   1,
      "number_of_replicas": 0,
      "index.lifecycle.name":          "listen-stream-30d-policy",
      "index.lifecycle.rollover_alias": "listen-stream-logs",
      "index.mapping.total_fields.limit": 2000
    },
    "mappings": {
      "dynamic": true,
      "_source": { "enabled": true },
      "properties": {
        "@timestamp":    { "type": "date" },
        "level":         { "type": "keyword" },
        "message":       { "type": "text", "fields": { "keyword": { "type": "keyword", "ignore_above": 256 } } },
        "environment":   { "type": "keyword" },
        "cluster":       { "type": "keyword" },
        "service": {
          "properties": {
            "name":    { "type": "keyword" },
            "version": { "type": "keyword" }
          }
        },
        "trace": {
          "properties": {
            "id": { "type": "keyword" }
          }
        },
        "span": {
          "properties": {
            "id": { "type": "keyword" }
          }
        },
        "request": {
          "properties": {
            "id": { "type": "keyword" }
          }
        },
        "user": {
          "properties": {
            "id": { "type": "keyword" }
          }
        },
        "http": {
          "properties": {
            "method":   { "type": "keyword" },
            "response": { "properties": { "status_code": { "type": "integer" } } }
          }
        },
        "url": {
          "properties": {
            "path": { "type": "keyword" }
          }
        },
        "event": {
          "properties": {
            "duration":    { "type": "long" },
            "duration_ms": { "type": "float" }
          }
        },
        "error": {
          "properties": {
            "message": { "type": "text" }
          }
        },
        "client": {
          "properties": {
            "ip":  { "type": "ip" },
            "geo": { "properties": { "country_name": { "type": "keyword" }, "city_name": { "type": "keyword" } } }
          }
        }
      }
    }
  }
}' > /dev/null
echo "  ✓ Index template created"

# ── 3. Error logs template ────────────────────────────────────────────────────
echo "==> Creating error index template..."
curl -sf -X PUT "${ES_URL}/_index_template/listen-stream-errors" \
  -H "Content-Type: application/json" -d '{
  "index_patterns": ["listen-stream-errors-*"],
  "priority": 200,
  "template": {
    "settings": {
      "number_of_shards": 1,
      "number_of_replicas": 0,
      "index.lifecycle.name": "listen-stream-30d-policy"
    }
  }
}' > /dev/null
echo "  ✓ Error index template created"

# ── 4. Write alias bootstrapping ─────────────────────────────────────────────
echo "==> Bootstrapping initial index and alias..."
curl -sf -X PUT "${ES_URL}/listen-stream-logs-bootstrap-000001" \
  -H "Content-Type: application/json" -d '{
  "aliases": {
    "listen-stream-logs": {
      "is_write_index": true
    }
  }
}' > /dev/null 2>&1 || echo "  (alias may already exist — skipping)"
echo "  ✓ Bootstrap index ready"

echo ""
echo "==> ✓ Elasticsearch initialized successfully!"
echo "==> Browse at: ${ES_URL}/_cat/indices?v"
