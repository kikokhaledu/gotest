function defaultKeyGenerator(req) {
  return req.ip || req.socket?.remoteAddress || "unknown";
}

function createRateLimiter({
  enabled = true,
  windowMs,
  maxRequests,
  keyGenerator = defaultKeyGenerator,
} = {}) {
  if (!enabled) {
    return (req, res, next) => next();
  }

  const buckets = new Map();
  let lastCleanupAt = 0;

  return (req, res, next) => {
    const now = Date.now();
    const key = keyGenerator(req);
    const bucketKey = `${key}`;

    if (now - lastCleanupAt > windowMs) {
      for (const [storedKey, bucket] of buckets.entries()) {
        if (bucket.resetAt <= now) {
          buckets.delete(storedKey);
        }
      }
      lastCleanupAt = now;
    }

    let bucket = buckets.get(bucketKey);
    if (!bucket || bucket.resetAt <= now) {
      bucket = {
        count: 0,
        resetAt: now + windowMs,
      };
      buckets.set(bucketKey, bucket);
    }

    bucket.count += 1;
    const remaining = Math.max(maxRequests - bucket.count, 0);
    const retryAfterSeconds = Math.max(Math.ceil((bucket.resetAt - now) / 1000), 1);

    res.setHeader("X-RateLimit-Limit", String(maxRequests));
    res.setHeader("X-RateLimit-Remaining", String(remaining));
    res.setHeader("X-RateLimit-Reset", String(Math.ceil(bucket.resetAt / 1000)));

    if (bucket.count > maxRequests) {
      res.setHeader("Retry-After", String(retryAfterSeconds));
      res.status(429).json({
        error: `Rate limit exceeded. Try again in ${retryAfterSeconds}s.`,
      });
      return;
    }

    next();
  };
}

module.exports = {
  createRateLimiter,
};
