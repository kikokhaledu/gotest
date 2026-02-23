const express = require('express');
const router = express.Router();

const config = require("../config");
const { authenticateApiKey } = require("../middleware/auth");
const { createRateLimiter } = require("../middleware/rateLimit");
const healthRoutes = require('./health');
const authRoutes = require("./auth");
const userRoutes = require('./users');
const taskRoutes = require('./tasks');
const statsRoutes = require('./stats');

const authRateLimiter = createRateLimiter({
  enabled: config.RATE_LIMIT_ENABLED,
  windowMs: config.RATE_LIMIT_WINDOW_MS,
  maxRequests: config.AUTH_RATE_LIMIT_MAX_REQUESTS,
});

const apiRateLimiter = createRateLimiter({
  enabled: config.RATE_LIMIT_ENABLED,
  windowMs: config.RATE_LIMIT_WINDOW_MS,
  maxRequests: config.RATE_LIMIT_MAX_REQUESTS,
});

const apiRoutes = express.Router();
apiRoutes.use("/users", userRoutes);
apiRoutes.use("/tasks", taskRoutes);
apiRoutes.use("/stats", statsRoutes);

// Mount routes
router.use('/', healthRoutes);
router.use("/auth", authRateLimiter, authRoutes);
router.use("/api", apiRateLimiter, authenticateApiKey, apiRoutes);


module.exports = router;
