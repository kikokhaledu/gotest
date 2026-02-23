import axios from 'axios'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:3000'
const DEFAULT_USERNAME = import.meta.env.VITE_AUTH_USERNAME || 'admin'
const DEFAULT_PASSWORD = import.meta.env.VITE_AUTH_PASSWORD || 'ChangeMe123@'
const DEFAULT_API_KEY = import.meta.env.VITE_API_KEY || ''
const DEFAULT_ACTOR = import.meta.env.VITE_ACTOR_NAME || DEFAULT_USERNAME

let cachedApiKey = DEFAULT_API_KEY.trim() || null
let loginInFlight = null

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

const isAuthExemptPath = (path = '') => {
  return path.startsWith('/health') || path.startsWith('/auth/login')
}

const ensureApiKey = async () => {
  if (cachedApiKey) {
    return cachedApiKey
  }

  if (loginInFlight) {
    return loginInFlight
  }

  loginInFlight = axios
    .post(
      `${API_BASE_URL}/auth/login`,
      {
        username: DEFAULT_USERNAME,
        password: DEFAULT_PASSWORD,
      },
      {
        timeout: 10000,
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )
    .then((response) => {
      const apiKey = response?.data?.apiKey
      if (!apiKey) {
        throw new Error('Login succeeded but no API key was returned')
      }
      cachedApiKey = apiKey
      return apiKey
    })
    .catch((error) => {
      cachedApiKey = null
      if (error.response) {
        throw new Error(error.response.data?.error || 'Authentication failed')
      }
      throw new Error('Authentication failed: unable to reach Node backend')
    })
    .finally(() => {
      loginInFlight = null
    })

  return loginInFlight
}

// Request interceptor for logging
apiClient.interceptors.request.use(
  async (config) => {
    const method = config.method?.toUpperCase() || 'GET'
    const path = config.url || ''
    console.log(`Making ${method} request to ${path}`)

    if (!isAuthExemptPath(path)) {
      const apiKey = await ensureApiKey()
      config.headers = config.headers || {}
      config.headers['x-api-key'] = apiKey
      config.headers['x-actor'] = DEFAULT_ACTOR
    }

    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor for error handling
apiClient.interceptors.response.use(
  (response) => response,
  async (error) => {
    const requestPath = error.config?.url || ''
    if (
      error.response?.status === 401 &&
      !isAuthExemptPath(requestPath) &&
      error.config?._retry !== true
    ) {
      try {
        cachedApiKey = null
        const apiKey = await ensureApiKey()
        error.config._retry = true
        error.config.headers = error.config.headers || {}
        error.config.headers['x-api-key'] = apiKey
        return apiClient.request(error.config)
      } catch (authError) {
        throw authError
      }
    }

    if (error.response) {
      // Server responded with error status
      throw new Error(
        error.response.data?.error || 
        error.response.data?.message || 
        `Request failed with status ${error.response.status}`
      )
    } else if (error.request) {
      // Request was made but no response received
      throw new Error('No response from server. Is the Node.js backend running?')
    } else {
      // Something else happened
      throw new Error(error.message || 'An unexpected error occurred')
    }
  }
)

export const checkHealth = async () => {
  try {
    const response = await apiClient.get('/health')
    return response.data
  } catch (error) {
    throw new Error(`Health check failed: ${error.message}`)
  }
}

export const getUsers = async () => {
  const response = await apiClient.get('/api/users')
  return response.data
}

export const getUserById = async (id) => {
  const response = await apiClient.get(`/api/users/${id}`)
  return response.data
}

export const getTasks = async (status = '', userId = '') => {
  const params = {}
  if (status) params.status = status
  if (userId) params.userId = userId
  
  const response = await apiClient.get('/api/tasks', { params })
  return response.data
}

export const getStats = async () => {
  const response = await apiClient.get('/api/stats')
  return response.data
}

export const getTaskHistory = async (id) => {
  const response = await apiClient.get(`/api/tasks/${id}/history`)
  return response.data
}

export const createUser = async (userData) => {
  const response = await apiClient.post('/api/users', userData)
  return response.data
}

export const createTask = async (taskData) => {
  const response = await apiClient.post('/api/tasks', taskData)
  return response.data
}

export const updateTask = async (id, taskData) => {
  const response = await apiClient.put(`/api/tasks/${id}`, taskData)
  return response.data
}
