const { makeRequest } = require('../utils/httpClient');

function normalizeStatusCode(error, fallback = 500) {
  const statusCode = error?.statusCode;
  if (Number.isInteger(statusCode) && statusCode >= 400 && statusCode <= 599) {
    return statusCode;
  }
  return fallback;
}

/**
 * Get all users
 */
const getAllUsers = async (req, res) => {
  try {
    const response = await makeRequest('/api/users');
    res.json(response);
  } catch (error) {
    const statusCode = normalizeStatusCode(error, 500);
    res.status(statusCode).json({ error: error.message || 'Internal server error' });
  }
};

/**
 * Get user by ID
 */
const getUserById = async (req, res) => {
  try {
    const user = await makeRequest(`/api/users/${req.params.id}`);
    res.json(user);
  } catch (error) {
    const statusCode = normalizeStatusCode(error, 500);
    if (statusCode === 404) {
      res.status(404).json({ error: 'User not found' });
    } else {
      res.status(statusCode).json({ error: error.message || 'Internal server error' });
    }
  }
};

/**
 * Create a new user
 */
const createUser = async (req, res) => {
  try {
    const response = await makeRequest('/api/users', {
      method: 'POST',
      body: req.body
    });
    res.status(201).json(response);
  } catch (error) {
    const statusCode = normalizeStatusCode(error, 500);
    res.status(statusCode).json({ error: error.message || 'Internal server error' });
  }
};

module.exports = {
  getAllUsers,
  getUserById,
  createUser
};
