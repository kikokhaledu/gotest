const http = require('http');
const config = require('../config');

const DEFAULT_REQUEST_TIMEOUT_MS = config.GO_REQUEST_TIMEOUT_MS;
const DEFAULT_MAX_RESPONSE_BYTES = config.GO_MAX_RESPONSE_BYTES;

function createError(message, statusCode, code) {
  const err = new Error(message);
  if (statusCode !== undefined) {
    err.statusCode = statusCode;
  }
  if (code) {
    err.code = code;
  }
  return err;
}

/**
 * Helper function to make HTTP requests to Go backend
 * @param {string} path - API endpoint path
 * @param {object} options - Request options (method, headers, body, timeoutMs, maxResponseBytes, signal)
 * @returns {Promise} - Resolves with response data or rejects with error
 */
function makeRequest(path, options = {}) {
  return new Promise((resolve, reject) => {
    const url = new URL(path, config.GO_BACKEND_URL);
    const timeoutMs = options.timeoutMs || DEFAULT_REQUEST_TIMEOUT_MS;
    const maxResponseBytes = options.maxResponseBytes || DEFAULT_MAX_RESPONSE_BYTES;
    const requestOptions = {
      hostname: url.hostname,
      port: url.port || 8080,
      path: url.pathname + url.search,
      method: options.method || 'GET',
      headers: {
        'Content-Type': 'application/json',
        ...options.headers
      }
    };

    if (options.signal?.aborted) {
      reject(createError('Request was aborted before start', 499, 'ERR_REQUEST_ABORTED'));
      return;
    }

    let settled = false;
    let abortListener = null;
    const settle = (callback, payload) => {
      if (settled) {
        return;
      }
      settled = true;

      if (abortListener && options.signal) {
        options.signal.removeEventListener('abort', abortListener);
      }

      callback(payload);
    };

    const req = http.request(requestOptions, (res) => {
      const chunks = [];
      let totalBytes = 0;

      res.on('error', (error) => {
        settle(reject, error);
      });

      res.on('data', (chunk) => {
        totalBytes += chunk.length;
        if (totalBytes > maxResponseBytes) {
          const sizeErr = createError(
            `Go backend response too large (>${maxResponseBytes} bytes)`,
            502,
            'ERR_RESPONSE_TOO_LARGE'
          );
          settle(reject, sizeErr);
          res.destroy();
          req.destroy();
          return;
        }
        chunks.push(chunk);
      });

      res.on('aborted', () => {
        settle(reject, createError('Go backend response aborted', 502, 'ERR_UPSTREAM_ABORTED'));
      });

      res.on('end', () => {
        const data = Buffer.concat(chunks).toString('utf8');
        if (res.statusCode >= 200 && res.statusCode < 300) {
          try {
            settle(resolve, JSON.parse(data));
          } catch (e) {
            settle(resolve, data);
          }
        } else {
          // Try to parse error response as JSON
          let errorMessage = data;
          try {
            const errorData = JSON.parse(data);
            errorMessage = errorData.error || errorData.message || data;
          } catch (e) {
            // Keep original error message if not JSON
          }
          const error = new Error(errorMessage);
          error.statusCode = res.statusCode;
          error.code = 'ERR_UPSTREAM_STATUS';
          error.responseData = data;
          settle(reject, error);
        }
      });
    });

    req.on('error', (error) => {
      settle(reject, error);
    });

    req.setTimeout(timeoutMs, () => {
      settle(reject, createError(`Go backend request timed out after ${timeoutMs}ms`, 504, 'ERR_UPSTREAM_TIMEOUT'));
      req.destroy();
    });

    if (options.signal) {
      abortListener = () => {
        settle(reject, createError('Request aborted', 499, 'ERR_REQUEST_ABORTED'));
        req.destroy();
      };
      options.signal.addEventListener('abort', abortListener, { once: true });
    }

    if (options.body) {
      req.write(JSON.stringify(options.body));
    }

    req.end();
  });
}

module.exports = { makeRequest };
