const test = require("node:test");
const assert = require("node:assert/strict");
const path = require("node:path");

const configPath = path.resolve(__dirname, "..", "config", "index.js");
const authPath = path.resolve(__dirname, "..", "middleware", "auth.js");
const relevantEnvKeys = ["AUTH_ENABLED", "AUTH_USERNAME", "AUTH_PASSWORD", "AUTH_API_KEY"];

function clearModuleCache() {
	delete require.cache[configPath];
	delete require.cache[authPath];
}

async function withEnv(overrides, fn) {
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

	clearModuleCache();
	try {
		return await fn();
	} finally {
		for (const key of relevantEnvKeys) {
			if (previousValues[key] === undefined) {
				delete process.env[key];
			} else {
				process.env[key] = previousValues[key];
			}
		}
		clearModuleCache();
	}
}

function makeReq(headers = {}) {
	const normalized = {};
	for (const [key, value] of Object.entries(headers)) {
		normalized[String(key).toLowerCase()] = value;
	}
	return {
		get(name) {
			return normalized[String(name).toLowerCase()];
		},
	};
}

function makeRes() {
	return {
		statusCode: null,
		body: null,
		status(code) {
			this.statusCode = code;
			return this;
		},
		json(payload) {
			this.body = payload;
			return this;
		},
	};
}

test("validateCredentials succeeds only for configured username/password", async () => {
	await withEnv(
		{
			AUTH_USERNAME: "admin",
			AUTH_PASSWORD: "ChangeMe123@",
		},
		async () => {
			const { validateCredentials } = require(authPath);

			assert.equal(validateCredentials("admin", "ChangeMe123@"), true);
			assert.equal(validateCredentials("admin", "wrong"), false);
			assert.equal(validateCredentials("wrong", "ChangeMe123@"), false);
			assert.equal(validateCredentials(undefined, "ChangeMe123@"), false);
			assert.equal(validateCredentials("admin", undefined), false);
		}
	);
});

test("authenticateApiKey returns 401 when API key is missing", async () => {
	await withEnv(
		{
			AUTH_ENABLED: "true",
			AUTH_API_KEY: "test-key",
		},
		async () => {
			const { authenticateApiKey } = require(authPath);
			const req = makeReq();
			const res = makeRes();
			let nextCalled = false;

			authenticateApiKey(req, res, () => {
				nextCalled = true;
			});

			assert.equal(nextCalled, false);
			assert.equal(res.statusCode, 401);
			assert.match(res.body.error, /missing api key/i);
		}
	);
});

test("authenticateApiKey returns 401 when x-api-key is invalid", async () => {
	await withEnv(
		{
			AUTH_ENABLED: "true",
			AUTH_API_KEY: "expected-key",
		},
		async () => {
			const { authenticateApiKey } = require(authPath);
			const req = makeReq({ "x-api-key": "wrong-key" });
			const res = makeRes();
			let nextCalled = false;

			authenticateApiKey(req, res, () => {
				nextCalled = true;
			});

			assert.equal(nextCalled, false);
			assert.equal(res.statusCode, 401);
			assert.match(res.body.error, /invalid api key/i);
		}
	);
});

test("authenticateApiKey accepts x-api-key and trims whitespace", async () => {
	await withEnv(
		{
			AUTH_ENABLED: "true",
			AUTH_API_KEY: "expected-key",
		},
		async () => {
			const { authenticateApiKey } = require(authPath);
			const req = makeReq({ "x-api-key": "   expected-key   " });
			const res = makeRes();
			let nextCalled = false;

			authenticateApiKey(req, res, () => {
				nextCalled = true;
			});

			assert.equal(nextCalled, true);
			assert.equal(res.statusCode, null);
		}
	);
});

test("authenticateApiKey accepts Authorization Bearer token", async () => {
	await withEnv(
		{
			AUTH_ENABLED: "true",
			AUTH_API_KEY: "bearer-key",
		},
		async () => {
			const { authenticateApiKey } = require(authPath);
			const req = makeReq({ authorization: "Bearer bearer-key" });
			const res = makeRes();
			let nextCalled = false;

			authenticateApiKey(req, res, () => {
				nextCalled = true;
			});

			assert.equal(nextCalled, true);
			assert.equal(res.statusCode, null);
		}
	);
});

test("authenticateApiKey prioritizes x-api-key over Authorization header", async () => {
	await withEnv(
		{
			AUTH_ENABLED: "true",
			AUTH_API_KEY: "expected-key",
		},
		async () => {
			const { authenticateApiKey } = require(authPath);
			const req = makeReq({
				"x-api-key": "wrong-key",
				authorization: "Bearer expected-key",
			});
			const res = makeRes();
			let nextCalled = false;

			authenticateApiKey(req, res, () => {
				nextCalled = true;
			});

			assert.equal(nextCalled, false);
			assert.equal(res.statusCode, 401);
			assert.match(res.body.error, /invalid api key/i);
		}
	);
});

test("authenticateApiKey bypasses checks when AUTH_ENABLED is false", async () => {
	await withEnv(
		{
			AUTH_ENABLED: "false",
			AUTH_API_KEY: "expected-key",
		},
		async () => {
			const { authenticateApiKey } = require(authPath);
			const req = makeReq();
			const res = makeRes();
			let nextCalled = false;

			authenticateApiKey(req, res, () => {
				nextCalled = true;
			});

			assert.equal(nextCalled, true);
			assert.equal(res.statusCode, null);
		}
	);
});
