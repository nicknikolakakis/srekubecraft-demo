"""
App1 — A simple message classifier bot.
Built by a non-technical user with Claude's help.

This app uses Dapr for everything:
  - Secrets via Dapr secret store (backed by Vault — the app doesn't know)
  - State via Dapr state store (backed by Redis — the app doesn't know)
  - POST /classify  — classify a message and save it
  - GET /history    — list past classifications
  - GET /health     — Kubernetes health probe
"""

import json
import os
import uuid
from datetime import datetime, timezone

import requests
from flask import Flask, jsonify, request

app = Flask(__name__)

DAPR_PORT = os.getenv("DAPR_HTTP_PORT", "3500")
DAPR = f"http://localhost:{DAPR_PORT}/v1.0"
STATE_STORE = "app1-statestore"
SECRET_STORE = "app1-secretstore"


def get_secret(key):
    """Get a secret from Dapr secret store (Vault behind the scenes)."""
    resp = requests.get(f"{DAPR}/secrets/{SECRET_STORE}/{key}", timeout=5)
    if resp.status_code == 200:
        return resp.json()
    return {}


def classify_message(text):
    """Simple keyword-based classifier (placeholder for Claude API call).

    In production, this would call the Anthropic API using the api key
    retrieved from Dapr secrets. For the demo, we use keyword matching.
    """
    text_lower = text.lower()
    if any(w in text_lower for w in ["bug", "error", "broken", "crash", "fix"]):
        return "bug"
    elif any(w in text_lower for w in ["feature", "request", "add", "want", "need"]):
        return "feature_request"
    return "question"


@app.route("/health", methods=["GET"])
def health():
    return jsonify({"status": "healthy"}), 200


@app.route("/classify", methods=["POST"])
def classify():
    body = request.get_json(force=True)
    text = body.get("message", "")
    if not text:
        return jsonify({"error": "message field is required"}), 400

    classification = classify_message(text)
    entry_id = str(uuid.uuid4())[:8]

    # Save to Dapr state store
    state_entry = {
        "id": entry_id,
        "message": text,
        "classification": classification,
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }

    requests.post(
        f"{DAPR}/state/{STATE_STORE}",
        json=[{"key": entry_id, "value": state_entry}],
        timeout=5,
    )

    # Append to history list
    history_resp = requests.get(f"{DAPR}/state/{STATE_STORE}/history", timeout=5)
    history = history_resp.json() if history_resp.status_code == 200 and history_resp.content else []
    history.append(entry_id)

    requests.post(
        f"{DAPR}/state/{STATE_STORE}",
        json=[{"key": "history", "value": history}],
        timeout=5,
    )

    return jsonify(state_entry), 200


@app.route("/history", methods=["GET"])
def history():
    resp = requests.get(f"{DAPR}/state/{STATE_STORE}/history", timeout=5)
    if resp.status_code != 200 or not resp.content:
        return jsonify([]), 200

    ids = resp.json()
    entries = []
    for entry_id in ids[-10:]:
        entry_resp = requests.get(f"{DAPR}/state/{STATE_STORE}/{entry_id}", timeout=5)
        if entry_resp.status_code == 200 and entry_resp.content:
            entries.append(entry_resp.json())

    return jsonify(entries), 200


@app.route("/secrets-check", methods=["GET"])
def secrets_check():
    """Demo endpoint — shows that the app can read secrets via Dapr.
    Returns only key names, never values."""
    secrets = get_secret("app1")
    return jsonify({"available_keys": list(secrets.keys())}), 200


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=8080)
