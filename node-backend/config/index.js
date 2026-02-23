function parsePositiveInt(value, fallback) {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return fallback;
  }
  return parsed;
}

function parseBoolean(value, fallback) {
  if (value === undefined) {
    return fallback;
  }

  const normalized = String(value).trim().toLowerCase();
  if (normalized === "true" || normalized === "1" || normalized === "yes" || normalized === "on") {
    return true;
  }
  if (normalized === "false" || normalized === "0" || normalized === "no" || normalized === "off") {
    return false;
  }
  return fallback;
}

module.exports = {
  PORT: process.env.PORT || 3000,
  GO_BACKEND_URL: process.env.GO_BACKEND_URL || 'http://localhost:8080',
  NODE_ENV: process.env.NODE_ENV || 'development',
  GO_REQUEST_TIMEOUT_MS: parsePositiveInt(process.env.GO_REQUEST_TIMEOUT_MS, 5000),
  GO_MAX_RESPONSE_BYTES: parsePositiveInt(process.env.GO_MAX_RESPONSE_BYTES, 1024 * 1024),
  SHUTDOWN_TIMEOUT_MS: parsePositiveInt(process.env.SHUTDOWN_TIMEOUT_MS, 10000),
  AUTH_ENABLED: parseBoolean(process.env.AUTH_ENABLED, true),
  AUTH_USERNAME: process.env.AUTH_USERNAME || "admin",
  AUTH_PASSWORD: process.env.AUTH_PASSWORD || "ChangeMe123@",
  AUTH_API_KEY: process.env.AUTH_API_KEY || "dev-local-api-key",
  RATE_LIMIT_ENABLED: parseBoolean(process.env.RATE_LIMIT_ENABLED, true),
  RATE_LIMIT_WINDOW_MS: parsePositiveInt(process.env.RATE_LIMIT_WINDOW_MS, 60000),
  RATE_LIMIT_MAX_REQUESTS: parsePositiveInt(process.env.RATE_LIMIT_MAX_REQUESTS, 120),
  AUTH_RATE_LIMIT_MAX_REQUESTS: parsePositiveInt(process.env.AUTH_RATE_LIMIT_MAX_REQUESTS, 10),
};
