const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const path = require("node:path");

const appPath = path.resolve(__dirname, "..", "app.js");
const configPath = path.resolve(__dirname, "..", "config", "index.js");
const backendRoot = path.resolve(__dirname, "..");
const relevantEnvKeys = [
  "GO_BACKEND_URL",
  "AUTH_ENABLED",
  "AUTH_USERNAME",
  "AUTH_PASSWORD",
  "AUTH_API_KEY",
  "RATE_LIMIT_ENABLED",
  "RATE_LIMIT_WINDOW_MS",
  "RATE_LIMIT_MAX_REQUESTS",
  "AUTH_RATE_LIMIT_MAX_REQUESTS",
];

function clearModuleCache() {
  for (const modulePath of Object.keys(require.cache)) {
    if (modulePath.startsWith(backendRoot)) {
      delete require.cache[modulePath];
    }
  }

  delete require.cache[appPath];
  delete require.cache[configPath];
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

async function startNodeServer() {
  const { startServer } = require(appPath);
  const noopLogger = {
    log() {},
    error() {},
  };

  const server = startServer({
    port: 0,
    logger: noopLogger,
    installSignalHandlers: false,
  });

  await new Promise((resolve, reject) => {
    server.once("listening", resolve);
    server.once("error", reject);
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

async function requestJson({
  port,
  method = "GET",
  path = "/",
  headers = {},
  body,
}) {
  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        hostname: "127.0.0.1",
        port,
        method,
        path,
        headers,
      },
      (res) => {
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => {
          const rawBody = Buffer.concat(chunks).toString("utf8");
          let parsedBody = rawBody;
          try {
            parsedBody = rawBody ? JSON.parse(rawBody) : {};
          } catch (e) {
            // keep raw body for debugging
          }
          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            body: parsedBody,
          });
        });
      }
    );

    req.on("error", reject);

    if (body !== undefined) {
      req.write(JSON.stringify(body));
    }

    req.end();
  });
}

test("auth flow requires API key and allows access after login", async () => {
  const goServer = await startServer((req, res) => {
    if (req.url === "/health") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ status: "ok" }));
      return;
    }

    if (req.url === "/api/users") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ users: [{ id: 1, name: "Alice" }] }));
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "not found" }));
  });

  try {
    const goPort = goServer.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${goPort}`,
        AUTH_ENABLED: "true",
        AUTH_USERNAME: "admin",
        AUTH_PASSWORD: "ChangeMe123@",
        AUTH_API_KEY: "test-api-key",
        RATE_LIMIT_ENABLED: "true",
        RATE_LIMIT_WINDOW_MS: "60000",
        RATE_LIMIT_MAX_REQUESTS: "50",
        AUTH_RATE_LIMIT_MAX_REQUESTS: "10",
      },
      async () => {
        const nodeServer = await startNodeServer();
        try {
          const nodePort = nodeServer.address().port;

          const unauthorizedResponse = await requestJson({
            port: nodePort,
            path: "/api/users",
          });
          assert.equal(unauthorizedResponse.statusCode, 401);

          const loginResponse = await requestJson({
            port: nodePort,
            method: "POST",
            path: "/auth/login",
            headers: { "Content-Type": "application/json" },
            body: {
              username: "admin",
              password: "ChangeMe123@",
            },
          });

          assert.equal(loginResponse.statusCode, 200);
          assert.equal(loginResponse.body.apiKey, "test-api-key");

          const authorizedResponse = await requestJson({
            port: nodePort,
            path: "/api/users",
            headers: { "x-api-key": loginResponse.body.apiKey },
          });

          assert.equal(authorizedResponse.statusCode, 200);
          assert.deepEqual(authorizedResponse.body.users, [{ id: 1, name: "Alice" }]);
        } finally {
          await closeServer(nodeServer);
        }
      }
    );
  } finally {
    await closeServer(goServer);
  }
});

test("api rate limiting returns 429 when request budget is exceeded", async () => {
  const goServer = await startServer((req, res) => {
    if (req.url === "/health") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ status: "ok" }));
      return;
    }

    if (req.url === "/api/users") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ users: [] }));
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "not found" }));
  });

  try {
    const goPort = goServer.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${goPort}`,
        AUTH_ENABLED: "true",
        AUTH_USERNAME: "admin",
        AUTH_PASSWORD: "ChangeMe123@",
        AUTH_API_KEY: "rate-limit-test-key",
        RATE_LIMIT_ENABLED: "true",
        RATE_LIMIT_WINDOW_MS: "10000",
        RATE_LIMIT_MAX_REQUESTS: "2",
        AUTH_RATE_LIMIT_MAX_REQUESTS: "10",
      },
      async () => {
        const nodeServer = await startNodeServer();
        try {
          const nodePort = nodeServer.address().port;
          const headers = { "x-api-key": "rate-limit-test-key" };

          const firstResponse = await requestJson({
            port: nodePort,
            path: "/api/users",
            headers,
          });
          const secondResponse = await requestJson({
            port: nodePort,
            path: "/api/users",
            headers,
          });
          const thirdResponse = await requestJson({
            port: nodePort,
            path: "/api/users",
            headers,
          });

          assert.equal(firstResponse.statusCode, 200);
          assert.equal(secondResponse.statusCode, 200);
          assert.equal(thirdResponse.statusCode, 429);
          assert.ok(thirdResponse.headers["retry-after"]);
        } finally {
          await closeServer(nodeServer);
        }
      }
    );
  } finally {
    await closeServer(goServer);
  }
});

test("auth login validates payload and credentials", async () => {
  const goServer = await startServer((req, res) => {
    if (req.url === "/health") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ status: "ok" }));
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "not found" }));
  });

  try {
    const goPort = goServer.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${goPort}`,
        AUTH_ENABLED: "true",
        AUTH_USERNAME: "admin",
        AUTH_PASSWORD: "ChangeMe123@",
        AUTH_API_KEY: "login-test-key",
        RATE_LIMIT_ENABLED: "false",
      },
      async () => {
        const nodeServer = await startNodeServer();
        try {
          const nodePort = nodeServer.address().port;

          const missingFieldsResponse = await requestJson({
            port: nodePort,
            method: "POST",
            path: "/auth/login",
            headers: { "Content-Type": "application/json" },
            body: {},
          });
          assert.equal(missingFieldsResponse.statusCode, 400);
          assert.match(missingFieldsResponse.body.error, /required/i);

          const invalidCredentialsResponse = await requestJson({
            port: nodePort,
            method: "POST",
            path: "/auth/login",
            headers: { "Content-Type": "application/json" },
            body: {
              username: "admin",
              password: "wrong",
            },
          });
          assert.equal(invalidCredentialsResponse.statusCode, 401);
          assert.match(invalidCredentialsResponse.body.error, /invalid username or password/i);
        } finally {
          await closeServer(nodeServer);
        }
      }
    );
  } finally {
    await closeServer(goServer);
  }
});

test("auth middleware accepts Authorization Bearer token", async () => {
  const goServer = await startServer((req, res) => {
    if (req.url === "/health") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ status: "ok" }));
      return;
    }

    if (req.url === "/api/users") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ users: [{ id: 1, name: "Bearer User" }] }));
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "not found" }));
  });

  try {
    const goPort = goServer.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${goPort}`,
        AUTH_ENABLED: "true",
        AUTH_USERNAME: "admin",
        AUTH_PASSWORD: "ChangeMe123@",
        AUTH_API_KEY: "bearer-test-key",
        RATE_LIMIT_ENABLED: "false",
      },
      async () => {
        const nodeServer = await startNodeServer();
        try {
          const nodePort = nodeServer.address().port;

          const response = await requestJson({
            port: nodePort,
            path: "/api/users",
            headers: {
              Authorization: "Bearer bearer-test-key",
            },
          });

          assert.equal(response.statusCode, 200);
          assert.deepEqual(response.body.users, [{ id: 1, name: "Bearer User" }]);
        } finally {
          await closeServer(nodeServer);
        }
      }
    );
  } finally {
    await closeServer(goServer);
  }
});

test("auth can be disabled for API routes", async () => {
  const goServer = await startServer((req, res) => {
    if (req.url === "/health") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ status: "ok" }));
      return;
    }

    if (req.url === "/api/users") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ users: [{ id: 1, name: "Open Access User" }] }));
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "not found" }));
  });

  try {
    const goPort = goServer.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${goPort}`,
        AUTH_ENABLED: "false",
        AUTH_API_KEY: "ignored-when-auth-disabled",
        RATE_LIMIT_ENABLED: "false",
      },
      async () => {
        const nodeServer = await startNodeServer();
        try {
          const nodePort = nodeServer.address().port;
          const response = await requestJson({
            port: nodePort,
            path: "/api/users",
          });

          assert.equal(response.statusCode, 200);
          assert.deepEqual(response.body.users, [{ id: 1, name: "Open Access User" }]);
        } finally {
          await closeServer(nodeServer);
        }
      }
    );
  } finally {
    await closeServer(goServer);
  }
});

test("auth rate limiting protects /auth/login endpoint", async () => {
  const goServer = await startServer((req, res) => {
    if (req.url === "/health") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ status: "ok" }));
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "not found" }));
  });

  try {
    const goPort = goServer.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${goPort}`,
        AUTH_ENABLED: "true",
        AUTH_USERNAME: "admin",
        AUTH_PASSWORD: "ChangeMe123@",
        AUTH_API_KEY: "auth-rate-limit-key",
        RATE_LIMIT_ENABLED: "true",
        RATE_LIMIT_WINDOW_MS: "10000",
        AUTH_RATE_LIMIT_MAX_REQUESTS: "2",
        RATE_LIMIT_MAX_REQUESTS: "100",
      },
      async () => {
        const nodeServer = await startNodeServer();
        try {
          const nodePort = nodeServer.address().port;

          const firstResponse = await requestJson({
            port: nodePort,
            method: "POST",
            path: "/auth/login",
            headers: { "Content-Type": "application/json" },
            body: { username: "admin", password: "wrong" },
          });
          const secondResponse = await requestJson({
            port: nodePort,
            method: "POST",
            path: "/auth/login",
            headers: { "Content-Type": "application/json" },
            body: { username: "admin", password: "wrong" },
          });
          const thirdResponse = await requestJson({
            port: nodePort,
            method: "POST",
            path: "/auth/login",
            headers: { "Content-Type": "application/json" },
            body: { username: "admin", password: "wrong" },
          });

          assert.equal(firstResponse.statusCode, 401);
          assert.equal(secondResponse.statusCode, 401);
          assert.equal(thirdResponse.statusCode, 429);
          assert.ok(thirdResponse.headers["retry-after"]);
        } finally {
          await closeServer(nodeServer);
        }
      }
    );
  } finally {
    await closeServer(goServer);
  }
});

test("task updates forward actor identity and history endpoint is proxied", async () => {
  let capturedActor = "";

  const goServer = await startServer((req, res) => {
    if (req.url === "/health") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ status: "ok" }));
      return;
    }

    if (req.url === "/api/tasks/1" && req.method === "PUT") {
      capturedActor = String(req.headers["x-actor"] || "");
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify({
          id: 1,
          title: "Task",
          status: "completed",
          userId: 1,
          lastChange: {
            id: 10,
            taskId: 1,
            changedAt: "2026-02-23T10:00:00Z",
            changedBy: capturedActor,
            field: "status",
            fromValue: "pending",
            toValue: "completed",
          },
        })
      );
      return;
    }

    if (req.url === "/api/tasks/1/history" && req.method === "GET") {
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify({
          taskId: 1,
          count: 1,
          history: [
            {
              id: 10,
              taskId: 1,
              changedAt: "2026-02-23T10:00:00Z",
              changedBy: "qa-user",
              field: "status",
              fromValue: "pending",
              toValue: "completed",
            },
          ],
        })
      );
      return;
    }

    res.writeHead(404, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ error: "not found" }));
  });

  try {
    const goPort = goServer.address().port;
    await withEnv(
      {
        GO_BACKEND_URL: `http://127.0.0.1:${goPort}`,
        AUTH_ENABLED: "true",
        AUTH_USERNAME: "admin",
        AUTH_PASSWORD: "ChangeMe123@",
        AUTH_API_KEY: "history-test-key",
        RATE_LIMIT_ENABLED: "false",
      },
      async () => {
        const nodeServer = await startNodeServer();
        try {
          const nodePort = nodeServer.address().port;
          const headers = {
            "x-api-key": "history-test-key",
            "x-actor": "qa-user",
            "Content-Type": "application/json",
          };

          const updateResponse = await requestJson({
            port: nodePort,
            method: "PUT",
            path: "/api/tasks/1",
            headers,
            body: { status: "completed" },
          });
          assert.equal(updateResponse.statusCode, 200);
          assert.equal(capturedActor, "qa-user");

          const historyResponse = await requestJson({
            port: nodePort,
            path: "/api/tasks/1/history",
            headers: { "x-api-key": "history-test-key" },
          });
          assert.equal(historyResponse.statusCode, 200);
          assert.equal(historyResponse.body.count, 1);
          assert.equal(historyResponse.body.history[0].field, "status");
        } finally {
          await closeServer(nodeServer);
        }
      }
    );
  } finally {
    await closeServer(goServer);
  }
});
