const { makeRequest } = require('../utils/httpClient');

function normalizeStatusCode(error, fallback = 500) {
  const statusCode = error?.statusCode;
  if (Number.isInteger(statusCode) && statusCode >= 400 && statusCode <= 599) {
    return statusCode;
  }
  return fallback;
}

/**
 * Get all tasks with optional filters
 */
const getAllTasks = async (req, res) => {
  try {
    const { status, userId } = req.query;
    let path = '/api/tasks';
    const params = new URLSearchParams();
    if (status) params.append('status', status);
    if (userId) params.append('userId', userId);
    if (params.toString()) {
      path += '?' + params.toString();
    }
    const response = await makeRequest(path);
    res.json(response);
  } catch (error) {
    const statusCode = normalizeStatusCode(error, 500);
    res.status(statusCode).json({ error: error.message || 'Internal server error' });
  }
};

/**
 * Get full history for a task by ID
 */
const getTaskHistory = async (req, res) => {
  try {
    const response = await makeRequest(`/api/tasks/${req.params.id}/history`);
    res.json(response);
  } catch (error) {
    const statusCode = normalizeStatusCode(error, 500);
    res.status(statusCode).json({ error: error.message || 'Internal server error' });
  }
};

/**
 * Create a new task
 */
const createTask = async (req, res) => {
  try {
    const actor = req.actor || 'system';
    const response = await makeRequest('/api/tasks', {
      method: 'POST',
      body: req.body,
      headers: {
        'X-Actor': actor,
      },
    });
    res.status(201).json(response);
  } catch (error) {
    const statusCode = normalizeStatusCode(error, 500);
    res.status(statusCode).json({ error: error.message || 'Internal server error' });
  }
};

/**
 * Update a task by ID
 */
const updateTask = async (req, res) => {
  try {
    const actor = req.actor || 'system';
    const response = await makeRequest(`/api/tasks/${req.params.id}`, {
      method: 'PUT',
      body: req.body,
      headers: {
        'X-Actor': actor,
      },
    });
    res.json(response);
  } catch (error) {
    const statusCode = normalizeStatusCode(error, 500);
    res.status(statusCode).json({ error: error.message || 'Internal server error' });
  }
};

module.exports = {
  getAllTasks,
  getTaskHistory,
  createTask,
  updateTask
};
