const test = require("node:test");
const assert = require("node:assert/strict");
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

test("startServer can start on an ephemeral port and close cleanly", async () => {
  clearAppCache();
  const { startServer } = require(appPath);

  const noopLogger = {
    log() {},
    error() {}
  };

  const server = startServer({
    port: 0,
    logger: noopLogger,
    installSignalHandlers: false
  });

  try {
    await new Promise((resolve, reject) => {
      server.once("listening", resolve);
      server.once("error", reject);
    });

    const address = server.address();
    assert.ok(address);
    assert.equal(typeof address.port, "number");
    assert.ok(address.port > 0);
  } finally {
    await closeServer(server);
  }
});
