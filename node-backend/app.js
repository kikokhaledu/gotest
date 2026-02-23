const express = require('express');
const cors = require('cors');
const swaggerUi = require("swagger-ui-express");
const config = require('./config');
const openApiSpec = require("./docs/openapi");
const routes = require('./routes');
const { errorHandler, notFoundHandler } = require('./middleware/errorHandler');

const app = express();

// Middleware
app.use(cors());
app.use(express.json());

app.get("/openapi.json", (req, res) => {
  res.json(openApiSpec);
});

app.use("/docs", swaggerUi.serve, swaggerUi.setup(openApiSpec, { explorer: true }));

// Routes
app.use(routes);

// Error handling
app.use(errorHandler);
app.use(notFoundHandler);

function installGracefulShutdown(server, {
  shutdownTimeoutMs = config.SHUTDOWN_TIMEOUT_MS,
  logger = console
} = {}) {
  let shuttingDown = false;

  const shutdown = (signal) => {
    if (shuttingDown) {
      return;
    }
    shuttingDown = true;

    logger.log(`[shutdown] received ${signal}, closing Node.js server...`);
    const forceTimer = setTimeout(() => {
      logger.error(`[shutdown] force exit after ${shutdownTimeoutMs}ms`);
      process.exit(1);
    }, shutdownTimeoutMs);
    forceTimer.unref?.();

    server.close((err) => {
      clearTimeout(forceTimer);
      if (err) {
        logger.error('[shutdown] failed to close server cleanly', err);
        process.exit(1);
      }
      logger.log('[shutdown] Node.js server closed');
      process.exit(0);
    });
  };

  process.on('SIGTERM', () => shutdown('SIGTERM'));
  process.on('SIGINT', () => shutdown('SIGINT'));
}

function startServer(options = {}) {
  const port = options.port ?? config.PORT;
  const logger = options.logger ?? console;
  const installSignalHandlers = options.installSignalHandlers !== false;
  const shutdownTimeoutMs = options.shutdownTimeoutMs ?? config.SHUTDOWN_TIMEOUT_MS;

  const server = app.listen(port, () => {
    logger.log(`Node.js backend server running on http://localhost:${port}`);
    logger.log(`Connecting to Go backend at ${config.GO_BACKEND_URL}`);
    logger.log(`Health check: http://localhost:${port}/health`);
  });

  if (installSignalHandlers) {
    installGracefulShutdown(server, { shutdownTimeoutMs, logger });
  }

  return server;
}

if (require.main === module) {
  startServer();
}

module.exports = { app, startServer, installGracefulShutdown };
