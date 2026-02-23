const test = require("node:test");
const assert = require("node:assert/strict");
const http = require("node:http");
const path = require("node:path");

const appPath = path.resolve(__dirname, "..", "app.js");

function clearAppCache() {
  delete require.cache[appPath];
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

async function request(port, requestPath) {
  return new Promise((resolve, reject) => {
    const req = http.request(
      {
        hostname: "127.0.0.1",
        port,
        method: "GET",
        path: requestPath,
      },
      (res) => {
        const chunks = [];
        res.on("data", (chunk) => chunks.push(chunk));
        res.on("end", () => {
          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            body: Buffer.concat(chunks).toString("utf8"),
          });
        });
      }
    );

    req.on("error", reject);
    req.end();
  });
}

test("openapi spec is served from /openapi.json", async () => {
  clearAppCache();
  const { startServer } = require(appPath);
  const noopLogger = { log() {}, error() {} };

  const server = startServer({
    port: 0,
    logger: noopLogger,
    installSignalHandlers: false,
  });

  try {
    await new Promise((resolve, reject) => {
      server.once("listening", resolve);
      server.once("error", reject);
    });

    const port = server.address().port;
    const response = await request(port, "/openapi.json");
    assert.equal(response.statusCode, 200);

    const payload = JSON.parse(response.body);
    assert.equal(payload.openapi, "3.0.3");
    assert.ok(payload.paths["/health"]);
    assert.ok(payload.paths["/auth/login"]);
    assert.ok(payload.paths["/api/users"]);
    assert.ok(payload.paths["/api/tasks/{id}/history"]);
  } finally {
    await closeServer(server);
  }
});

test("openapi schemas match runtime health and stats response shapes", async () => {
  clearAppCache();
  const { startServer } = require(appPath);
  const noopLogger = { log() {}, error() {} };

  const server = startServer({
    port: 0,
    logger: noopLogger,
    installSignalHandlers: false,
  });

  try {
    await new Promise((resolve, reject) => {
      server.once("listening", resolve);
      server.once("error", reject);
    });

    const port = server.address().port;
    const response = await request(port, "/openapi.json");
    assert.equal(response.statusCode, 200);

    const payload = JSON.parse(response.body);
    const statsSchema = payload?.components?.schemas?.StatsResponse;
    assert.ok(statsSchema?.properties?.users);
    assert.ok(statsSchema?.properties?.tasks);
    assert.equal(statsSchema?.properties?.totalUsers, undefined);
    assert.equal(statsSchema?.properties?.tasksByStatus, undefined);

    const healthSchema = payload?.components?.schemas?.HealthResponse;
    assert.ok(healthSchema?.properties?.goBackend);

    const taskSchema = payload?.components?.schemas?.Task;
    assert.ok(taskSchema?.properties?.lastChange);
  } finally {
    await closeServer(server);
  }
});

test("swagger ui is served from /docs/", async () => {
  clearAppCache();
  const { startServer } = require(appPath);
  const noopLogger = { log() {}, error() {} };

  const server = startServer({
    port: 0,
    logger: noopLogger,
    installSignalHandlers: false,
  });

  try {
    await new Promise((resolve, reject) => {
      server.once("listening", resolve);
      server.once("error", reject);
    });

    const port = server.address().port;
    const response = await request(port, "/docs/");
    assert.equal(response.statusCode, 200);
    assert.ok(response.headers["content-type"].includes("text/html"));
    assert.match(response.body, /swagger-ui/i);
  } finally {
    await closeServer(server);
  }
});
