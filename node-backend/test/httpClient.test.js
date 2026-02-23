const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const path = require("node:path");

const configPath = path.resolve(__dirname, "..", "config", "index.js");
const httpClientPath = path.resolve(__dirname, "..", "utils", "httpClient.js");
const relevantEnvKeys = [
  "GO_BACKEND_URL",
  "GO_REQUEST_TIMEOUT_MS",
  "GO_MAX_RESPONSE_BYTES"
];

function clearModuleCache() {
  delete require.cache[configPath];
  delete require.cache[httpClientPath];
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

async function startServer(handler) {
  const server = http.createServer(handler);
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  return server;
}

async function closeServer(server) {
  await new Promise((resolve, reject) => {
    server.close((err) => {
      if (err) {
        reject(err);
        return;
      }
      resolve();
    });
  });
}

test("makeRequest times out when upstream is too slow", async () => {
  const server = await startServer((req, res) => {
    setTimeout(() => {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true }));
    }, 120);
  });

  try {
    const port = server.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${port}`,
        GO_REQUEST_TIMEOUT_MS: "20",
        GO_MAX_RESPONSE_BYTES: "1048576"
      },
      async () => {
        const { makeRequest } = require(httpClientPath);
        await assert.rejects(
          () => makeRequest("/slow"),
          (err) => err?.code === "ERR_UPSTREAM_TIMEOUT" && err?.statusCode === 504
        );
      }
    );
  } finally {
    await closeServer(server);
  }
});

test("makeRequest rejects oversized upstream responses", async () => {
  const server = await startServer((req, res) => {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ payload: "x".repeat(2048) }));
  });

  try {
    const port = server.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${port}`,
        GO_REQUEST_TIMEOUT_MS: "2000",
        GO_MAX_RESPONSE_BYTES: "256"
      },
      async () => {
        const { makeRequest } = require(httpClientPath);
        await assert.rejects(
          () => makeRequest("/big"),
          (err) => err?.code === "ERR_RESPONSE_TOO_LARGE" && err?.statusCode === 502
        );
      }
    );
  } finally {
    await closeServer(server);
  }
});

test("makeRequest supports caller cancellation via AbortSignal", async () => {
  const server = await startServer((req, res) => {
    setTimeout(() => {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ ok: true }));
    }, 200);
  });

  try {
    const port = server.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${port}`,
        GO_REQUEST_TIMEOUT_MS: "1000",
        GO_MAX_RESPONSE_BYTES: "1048576"
      },
      async () => {
        const { makeRequest } = require(httpClientPath);
        const controller = new AbortController();
        setTimeout(() => controller.abort(), 20);

        await assert.rejects(
          () => makeRequest("/cancel", { signal: controller.signal }),
          (err) => err?.code === "ERR_REQUEST_ABORTED" && err?.statusCode === 499
        );
      }
    );
  } finally {
    await closeServer(server);
  }
});

test("makeRequest rejects immediately for pre-aborted signal", async () => {
  await withEnv(
    {
      GO_BACKEND_URL: "http://127.0.0.1:65535",
      GO_REQUEST_TIMEOUT_MS: "1000",
      GO_MAX_RESPONSE_BYTES: "1048576"
    },
    async () => {
      const { makeRequest } = require(httpClientPath);
      const controller = new AbortController();
      controller.abort();

      await assert.rejects(
        () => makeRequest("/never-sent", { signal: controller.signal }),
        (err) => err?.code === "ERR_REQUEST_ABORTED" && err?.statusCode === 499
      );
    }
  );
});

test("makeRequest preserves upstream status and JSON error message", async () => {
  const server = await startServer((req, res) => {
    res.writeHead(400, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "validation failed" }));
  });

  try {
    const port = server.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${port}`,
        GO_REQUEST_TIMEOUT_MS: "2000",
        GO_MAX_RESPONSE_BYTES: "1048576"
      },
      async () => {
        const { makeRequest } = require(httpClientPath);
        await assert.rejects(
          () => makeRequest("/bad-request"),
          (err) =>
            err?.code === "ERR_UPSTREAM_STATUS" &&
            err?.statusCode === 400 &&
            err?.message === "validation failed"
        );
      }
    );
  } finally {
    await closeServer(server);
  }
});

test("makeRequest returns raw text for non-JSON success payloads", async () => {
  const server = await startServer((req, res) => {
    res.writeHead(200, { "Content-Type": "text/plain" });
    res.end("plain-text-response");
  });

  try {
    const port = server.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${port}`,
        GO_REQUEST_TIMEOUT_MS: "2000",
        GO_MAX_RESPONSE_BYTES: "1048576"
      },
      async () => {
        const { makeRequest } = require(httpClientPath);
        const response = await makeRequest("/text");
        assert.equal(response, "plain-text-response");
      }
    );
  } finally {
    await closeServer(server);
  }
});
