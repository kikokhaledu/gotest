const test = require("node:test");
const assert = require("node:assert/strict");
const path = require("node:path");

const configPath = path.resolve(__dirname, "..", "config", "index.js");
const relevantEnvKeys = [
	"PORT",
	"GO_BACKEND_URL",
	"NODE_ENV",
	"GO_REQUEST_TIMEOUT_MS",
	"GO_MAX_RESPONSE_BYTES",
	"SHUTDOWN_TIMEOUT_MS",
	"AUTH_ENABLED",
	"AUTH_USERNAME",
	"AUTH_PASSWORD",
	"AUTH_API_KEY",
	"RATE_LIMIT_ENABLED",
	"RATE_LIMIT_WINDOW_MS",
	"RATE_LIMIT_MAX_REQUESTS",
	"AUTH_RATE_LIMIT_MAX_REQUESTS",
];

function withEnv(overrides, fn) {
	const previousValues = {};
	for (const key of relevantEnvKeys) {
		previousValues[key] = process.env[key];
	}

	for (const key of relevantEnvKeys) {
		if (Object.prototype.hasOwnProperty.call(overrides, key)) {
			const value = overrides[key];
			if (value === undefined) {
				delete process.env[key];
			} else {
				process.env[key] = value;
			}
		}
	}

	delete require.cache[configPath];

	try {
		return fn();
	} finally {
		for (const key of relevantEnvKeys) {
			if (previousValues[key] === undefined) {
				delete process.env[key];
			} else {
				process.env[key] = previousValues[key];
			}
		}
		delete require.cache[configPath];
	}
}

test("config uses documented defaults", () => {
	withEnv(
		{
			PORT: undefined,
			GO_BACKEND_URL: undefined,
			NODE_ENV: undefined,
			GO_REQUEST_TIMEOUT_MS: undefined,
			GO_MAX_RESPONSE_BYTES: undefined,
			SHUTDOWN_TIMEOUT_MS: undefined,
			AUTH_ENABLED: undefined,
			AUTH_USERNAME: undefined,
			AUTH_PASSWORD: undefined,
			AUTH_API_KEY: undefined,
			RATE_LIMIT_ENABLED: undefined,
			RATE_LIMIT_WINDOW_MS: undefined,
			RATE_LIMIT_MAX_REQUESTS: undefined,
			AUTH_RATE_LIMIT_MAX_REQUESTS: undefined,
		},
		() => {
			const config = require(configPath);
			assert.equal(config.PORT, 3000);
			assert.equal(config.GO_BACKEND_URL, "http://localhost:8080");
			assert.equal(config.NODE_ENV, "development");
			assert.equal(config.GO_REQUEST_TIMEOUT_MS, 5000);
			assert.equal(config.GO_MAX_RESPONSE_BYTES, 1024 * 1024);
			assert.equal(config.SHUTDOWN_TIMEOUT_MS, 10000);
			assert.equal(config.AUTH_ENABLED, true);
			assert.equal(config.AUTH_USERNAME, "admin");
			assert.equal(config.AUTH_PASSWORD, "ChangeMe123@");
			assert.equal(config.AUTH_API_KEY, "dev-local-api-key");
			assert.equal(config.RATE_LIMIT_ENABLED, true);
			assert.equal(config.RATE_LIMIT_WINDOW_MS, 60000);
			assert.equal(config.RATE_LIMIT_MAX_REQUESTS, 120);
			assert.equal(config.AUTH_RATE_LIMIT_MAX_REQUESTS, 10);
		}
	);
});

test("config respects environment overrides", () => {
	withEnv(
		{
			PORT: "3999",
			GO_BACKEND_URL: "http://go-backend:8080",
			NODE_ENV: "production",
			GO_REQUEST_TIMEOUT_MS: "7000",
			GO_MAX_RESPONSE_BYTES: "4096",
			SHUTDOWN_TIMEOUT_MS: "15000",
			AUTH_ENABLED: "false",
			AUTH_USERNAME: "qa-admin",
			AUTH_PASSWORD: "qa-pass",
			AUTH_API_KEY: "qa-key",
			RATE_LIMIT_ENABLED: "false",
			RATE_LIMIT_WINDOW_MS: "30000",
			RATE_LIMIT_MAX_REQUESTS: "77",
			AUTH_RATE_LIMIT_MAX_REQUESTS: "4",
		},
		() => {
			const config = require(configPath);
			assert.equal(config.PORT, "3999");
			assert.equal(config.GO_BACKEND_URL, "http://go-backend:8080");
			assert.equal(config.NODE_ENV, "production");
			assert.equal(config.GO_REQUEST_TIMEOUT_MS, 7000);
			assert.equal(config.GO_MAX_RESPONSE_BYTES, 4096);
			assert.equal(config.SHUTDOWN_TIMEOUT_MS, 15000);
			assert.equal(config.AUTH_ENABLED, false);
			assert.equal(config.AUTH_USERNAME, "qa-admin");
			assert.equal(config.AUTH_PASSWORD, "qa-pass");
			assert.equal(config.AUTH_API_KEY, "qa-key");
			assert.equal(config.RATE_LIMIT_ENABLED, false);
			assert.equal(config.RATE_LIMIT_WINDOW_MS, 30000);
			assert.equal(config.RATE_LIMIT_MAX_REQUESTS, 77);
			assert.equal(config.AUTH_RATE_LIMIT_MAX_REQUESTS, 4);
		}
	);
});

test("config falls back to defaults for invalid numeric values", () => {
	withEnv(
		{
			GO_REQUEST_TIMEOUT_MS: "0",
			GO_MAX_RESPONSE_BYTES: "-1",
			SHUTDOWN_TIMEOUT_MS: "not-a-number",
			RATE_LIMIT_WINDOW_MS: "0",
			RATE_LIMIT_MAX_REQUESTS: "-1",
			AUTH_RATE_LIMIT_MAX_REQUESTS: "x",
		},
		() => {
			const config = require(configPath);
			assert.equal(config.GO_REQUEST_TIMEOUT_MS, 5000);
			assert.equal(config.GO_MAX_RESPONSE_BYTES, 1024 * 1024);
			assert.equal(config.SHUTDOWN_TIMEOUT_MS, 10000);
			assert.equal(config.RATE_LIMIT_WINDOW_MS, 60000);
			assert.equal(config.RATE_LIMIT_MAX_REQUESTS, 120);
			assert.equal(config.AUTH_RATE_LIMIT_MAX_REQUESTS, 10);
		}
	);
});
