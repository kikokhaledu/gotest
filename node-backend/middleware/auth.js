const crypto = require("node:crypto");
const config = require("../config");

const MAX_ACTOR_LENGTH = 80;

function timingSafeEqual(left, right) {
  const leftBuffer = Buffer.from(String(left));
  const rightBuffer = Buffer.from(String(right));

  if (leftBuffer.length !== rightBuffer.length) {
    return false;
  }

  return crypto.timingSafeEqual(leftBuffer, rightBuffer);
}

function extractApiKey(req) {
  const apiKeyHeader = req.get("x-api-key");
  if (apiKeyHeader) {
    return apiKeyHeader.trim();
  }

  const authHeader = req.get("authorization");
  if (authHeader?.startsWith("Bearer ")) {
    return authHeader.slice("Bearer ".length).trim();
  }

  return "";
}

function validateCredentials(username, password) {
  return (
    timingSafeEqual(username ?? "", config.AUTH_USERNAME) &&
    timingSafeEqual(password ?? "", config.AUTH_PASSWORD)
  );
}

function resolveActor(req) {
  const actorFromHeader = req.get("x-actor")?.trim();
  if (actorFromHeader) {
    return actorFromHeader.slice(0, MAX_ACTOR_LENGTH);
  }

  const configuredUser = String(config.AUTH_USERNAME || "system").trim();
  if (configuredUser) {
    return configuredUser.slice(0, MAX_ACTOR_LENGTH);
  }

  return "system";
}

function authenticateApiKey(req, res, next) {
  if (!config.AUTH_ENABLED) {
    req.actor = resolveActor(req);
    next();
    return;
  }

  const apiKey = extractApiKey(req);
  if (!apiKey) {
    res.status(401).json({
      error: "Missing API key. Send x-api-key header or Authorization: Bearer <key>.",
    });
    return;
  }

  if (!timingSafeEqual(apiKey, config.AUTH_API_KEY)) {
    res.status(401).json({ error: "Invalid API key." });
    return;
  }

  req.actor = resolveActor(req);
  next();
}

module.exports = {
  authenticateApiKey,
  validateCredentials,
  resolveActor,
};
