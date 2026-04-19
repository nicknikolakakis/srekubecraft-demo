#!/bin/sh
# =============================================================================
# Seed demo secrets into Vault KV v2
# Runs inside the Vault pod via kubectl exec
# All values are dummy/demo — NOT real credentials
# =============================================================================
set -e

export VAULT_TOKEN=demo-root-token
export VAULT_ADDR=http://127.0.0.1:8200

vault kv put secret/dapr/app1 \
  slack-bot-token="xoxb-demo-token" \
  slack-app-token="xapp-demo-token" \
  anthropic-api-key="sk-ant-demo-key"

vault kv put secret/dapr/app2 \
  api-key="demo-app2-key" \
  db-connection="postgresql://demo:demo@follower-db:5432/app2"

echo "Demo secrets seeded successfully."
vault kv list secret/dapr/
