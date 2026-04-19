/**
 * App2 — Classification Dashboard
 * Built by a non-technical user with Claude's help.
 *
 * This app uses Dapr for everything:
 *   - Service invocation to call app1 (the classifier) via Dapr
 *   - State store to save dashboard preferences
 *   - Secrets to read API keys from Vault
 *
 * The citizen developer never imports Redis, Vault, or HTTP client libraries
 * for inter-service communication. Dapr handles it all via localhost:3500.
 */

import express from "express";
import path from "path";

const app = express();
const PORT = parseInt(process.env.PORT || "8080");
const DAPR_PORT = process.env.DAPR_HTTP_PORT || "3500";
const DAPR = `http://localhost:${DAPR_PORT}/v1.0`;
// Cross-namespace Dapr service invocation: app-id.namespace
const APP1_ID = "app1.app1-ns";

app.use(express.json());
app.use(express.static(path.join(__dirname, "../public")));

// Health check
app.get("/health", (_req, res) => {
  res.json({ status: "healthy" });
});

// Classify a message by invoking app1 via Dapr service invocation
// The citizen developer doesn't know app1's IP, port, or namespace.
// Dapr handles service discovery + mTLS automatically.
app.post("/api/classify", async (req, res) => {
  try {
    const response = await fetch(
      `${DAPR}/invoke/${APP1_ID}/method/classify`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ message: req.body.message }),
      }
    );
    const data = await response.json();
    res.json(data);
  } catch (err) {
    res.status(502).json({ error: "Failed to reach classifier service" });
  }
});

// Get classification history from app1 via Dapr service invocation
app.get("/api/history", async (_req, res) => {
  try {
    const response = await fetch(
      `${DAPR}/invoke/${APP1_ID}/method/history`
    );
    const data = await response.json();
    res.json(data);
  } catch (err) {
    res.status(502).json({ error: "Failed to reach classifier service" });
  }
});

// Check available secrets via Dapr secret store
app.get("/api/secrets-check", async (_req, res) => {
  try {
    const response = await fetch(
      `${DAPR}/secrets/app2-secretstore/app2`
    );
    const data = await response.json();
    res.json({ available_keys: Object.keys(data) });
  } catch (err) {
    res.json({ available_keys: [], note: "secret store not configured" });
  }
});

app.listen(PORT, "0.0.0.0", () => {
  console.log(`App2 dashboard listening on port ${PORT}`);
});
