const { makeRequest } = require('../utils/httpClient');

function normalizeStatusCode(error, fallback = 500) {
  const statusCode = error?.statusCode;
  if (Number.isInteger(statusCode) && statusCode >= 400 && statusCode <= 599) {
    return statusCode;
  }
  return fallback;
}

/**
 * Get statistics
 */
const getStats = async (req, res) => {
  try {
    const stats = await makeRequest('/api/stats');
    res.json(stats);
  } catch (error) {
    const statusCode = normalizeStatusCode(error, 500);
    res.status(statusCode).json({ error: error.message || 'Internal server error' });
  }
};

module.exports = {
  getStats
};
