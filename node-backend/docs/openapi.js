const openApiSpec = {
  openapi: "3.0.3",
  info: {
    title: "Go Test Node API Gateway",
    version: "1.0.0",
    description:
      "Node.js API gateway for the Go backend. Supports API-key auth and rate limiting.",
  },
  servers: [
    {
      url: "http://localhost:3000",
      description: "Local development",
    },
  ],
  tags: [
    { name: "Health", description: "Service health endpoints" },
    { name: "Auth", description: "Authentication endpoints" },
    { name: "Users", description: "User endpoints" },
    { name: "Tasks", description: "Task endpoints" },
    { name: "Stats", description: "Statistics endpoints" },
  ],
  components: {
    securitySchemes: {
      ApiKeyHeader: {
        type: "apiKey",
        in: "header",
        name: "x-api-key",
      },
      BearerAuth: {
        type: "http",
        scheme: "bearer",
        bearerFormat: "API key",
      },
    },
    schemas: {
      ErrorResponse: {
        type: "object",
        properties: {
          error: { type: "string" },
        },
      },
      HealthResponse: {
        type: "object",
        properties: {
          status: { type: "string", example: "ok" },
          message: { type: "string", example: "Node.js backend is running" },
          goBackend: { $ref: "#/components/schemas/GoHealthResponse" },
        },
      },
      GoHealthResponse: {
        type: "object",
        properties: {
          status: { type: "string", example: "ok" },
          message: { type: "string", example: "Go backend is running" },
        },
      },
      HealthUnavailableResponse: {
        type: "object",
        properties: {
          status: { type: "string", example: "error" },
          message: {
            type: "string",
            example: "Node.js backend is running but Go backend is unavailable",
          },
          error: { type: "string" },
        },
      },
      LoginRequest: {
        type: "object",
        required: ["username", "password"],
        properties: {
          username: { type: "string", example: "admin" },
          password: { type: "string", example: "ChangeMe123@" },
        },
      },
      LoginResponse: {
        type: "object",
        properties: {
          username: { type: "string", example: "admin" },
          tokenType: { type: "string", example: "ApiKey" },
          apiKey: { type: "string", example: "dev-local-api-key" },
        },
      },
      User: {
        type: "object",
        properties: {
          id: { type: "integer", example: 1 },
          name: { type: "string", example: "Alice" },
          email: { type: "string", example: "alice@example.com" },
          role: { type: "string", example: "developer" },
        },
      },
      Task: {
        type: "object",
        properties: {
          id: { type: "integer", example: 1 },
          title: { type: "string", example: "Build endpoint" },
          status: {
            type: "string",
            enum: ["pending", "in-progress", "completed"],
            example: "pending",
          },
          userId: { type: "integer", example: 1 },
          lastChange: {
            $ref: "#/components/schemas/TaskHistoryItem",
          },
        },
      },
      TaskHistoryItem: {
        type: "object",
        properties: {
          id: { type: "integer", example: 12 },
          taskId: { type: "integer", example: 1 },
          changedAt: { type: "string", format: "date-time", example: "2026-02-23T15:04:05Z" },
          changedBy: { type: "string", example: "admin" },
          field: { type: "string", enum: ["title", "status", "userId"], example: "status" },
          fromValue: { type: "string", nullable: true, example: "pending" },
          toValue: { type: "string", example: "in-progress" },
        },
      },
      TaskHistoryResponse: {
        type: "object",
        properties: {
          taskId: { type: "integer", example: 1 },
          history: {
            type: "array",
            items: { $ref: "#/components/schemas/TaskHistoryItem" },
          },
          count: { type: "integer", example: 2 },
        },
      },
      StatsResponse: {
        type: "object",
        properties: {
          users: {
            type: "object",
            properties: {
              total: { type: "integer", example: 3 },
            },
          },
          tasks: {
            type: "object",
            properties: {
              total: { type: "integer", example: 6 },
              pending: { type: "integer", example: 2 },
              inProgress: { type: "integer", example: 2 },
              completed: { type: "integer", example: 2 },
            },
          },
        },
      },
    },
  },
  paths: {
    "/health": {
      get: {
        tags: ["Health"],
        summary: "Node and upstream health status",
        responses: {
          200: {
            description: "Healthy",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/HealthResponse" },
              },
            },
          },
          503: {
            description: "Go backend unavailable",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/HealthUnavailableResponse" },
              },
            },
          },
        },
      },
    },
    "/auth/login": {
      post: {
        tags: ["Auth"],
        summary: "Authenticate and return API key",
        requestBody: {
          required: true,
          content: {
            "application/json": {
              schema: { $ref: "#/components/schemas/LoginRequest" },
            },
          },
        },
        responses: {
          200: {
            description: "Authenticated",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/LoginResponse" },
              },
            },
          },
          400: {
            description: "Invalid request",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
          401: {
            description: "Invalid credentials",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
          429: {
            description: "Rate limited",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
        },
      },
    },
    "/api/users": {
      get: {
        tags: ["Users"],
        summary: "Get all users",
        security: [{ ApiKeyHeader: [] }, { BearerAuth: [] }],
        responses: {
          200: {
            description: "User list",
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  properties: {
                    users: {
                      type: "array",
                      items: { $ref: "#/components/schemas/User" },
                    },
                  },
                },
              },
            },
          },
          401: {
            description: "Unauthorized",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
        },
      },
      post: {
        tags: ["Users"],
        summary: "Create user",
        security: [{ ApiKeyHeader: [] }, { BearerAuth: [] }],
        requestBody: {
          required: true,
          content: {
            "application/json": {
              schema: {
                type: "object",
                required: ["name", "email", "role"],
                properties: {
                  name: { type: "string" },
                  email: { type: "string" },
                  role: { type: "string" },
                },
              },
            },
          },
        },
        responses: {
          201: {
            description: "Created",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/User" },
              },
            },
          },
          400: {
            description: "Validation error",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
          401: {
            description: "Unauthorized",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
        },
      },
    },
    "/api/users/{id}": {
      get: {
        tags: ["Users"],
        summary: "Get user by ID",
        security: [{ ApiKeyHeader: [] }, { BearerAuth: [] }],
        parameters: [
          {
            name: "id",
            in: "path",
            required: true,
            schema: { type: "integer" },
          },
        ],
        responses: {
          200: {
            description: "User",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/User" },
              },
            },
          },
          401: {
            description: "Unauthorized",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
          404: {
            description: "Not found",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
        },
      },
    },
    "/api/tasks": {
      get: {
        tags: ["Tasks"],
        summary: "Get tasks",
        security: [{ ApiKeyHeader: [] }, { BearerAuth: [] }],
        parameters: [
          {
            name: "status",
            in: "query",
            required: false,
            schema: { type: "string" },
          },
          {
            name: "userId",
            in: "query",
            required: false,
            schema: { type: "integer" },
          },
        ],
        responses: {
          200: {
            description: "Task list",
            content: {
              "application/json": {
                schema: {
                  type: "object",
                  properties: {
                    tasks: {
                      type: "array",
                      items: { $ref: "#/components/schemas/Task" },
                    },
                  },
                },
              },
            },
          },
        },
      },
      post: {
        tags: ["Tasks"],
        summary: "Create task",
        security: [{ ApiKeyHeader: [] }, { BearerAuth: [] }],
        parameters: [
          {
            name: "x-actor",
            in: "header",
            required: false,
            schema: { type: "string" },
            description: "Optional actor name used in task history metadata.",
          },
        ],
        requestBody: {
          required: true,
          content: {
            "application/json": {
              schema: {
                type: "object",
                required: ["title", "status", "userId"],
                properties: {
                  title: { type: "string" },
                  status: {
                    type: "string",
                    enum: ["pending", "in-progress", "completed"],
                  },
                  userId: { type: "integer" },
                },
              },
            },
          },
        },
        responses: {
          201: {
            description: "Created",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/Task" },
              },
            },
          },
          400: {
            description: "Validation error",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
        },
      },
    },
    "/api/tasks/{id}": {
      put: {
        tags: ["Tasks"],
        summary: "Update task",
        security: [{ ApiKeyHeader: [] }, { BearerAuth: [] }],
        parameters: [
          {
            name: "id",
            in: "path",
            required: true,
            schema: { type: "integer" },
          },
          {
            name: "x-actor",
            in: "header",
            required: false,
            schema: { type: "string" },
            description: "Optional actor name used in task history metadata.",
          },
        ],
        requestBody: {
          required: true,
          content: {
            "application/json": {
              schema: {
                type: "object",
                properties: {
                  title: { type: "string" },
                  status: {
                    type: "string",
                    enum: ["pending", "in-progress", "completed"],
                  },
                  userId: { type: "integer" },
                },
              },
            },
          },
        },
        responses: {
          200: {
            description: "Updated",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/Task" },
              },
            },
          },
          400: {
            description: "Validation error",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
          404: {
            description: "Not found",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
        },
      },
    },
    "/api/tasks/{id}/history": {
      get: {
        tags: ["Tasks"],
        summary: "Get task history",
        security: [{ ApiKeyHeader: [] }, { BearerAuth: [] }],
        parameters: [
          {
            name: "id",
            in: "path",
            required: true,
            schema: { type: "integer" },
          },
        ],
        responses: {
          200: {
            description: "Task history",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/TaskHistoryResponse" },
              },
            },
          },
          404: {
            description: "Not found",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/ErrorResponse" },
              },
            },
          },
        },
      },
    },
    "/api/stats": {
      get: {
        tags: ["Stats"],
        summary: "Get aggregate stats",
        security: [{ ApiKeyHeader: [] }, { BearerAuth: [] }],
        responses: {
          200: {
            description: "Stats",
            content: {
              "application/json": {
                schema: { $ref: "#/components/schemas/StatsResponse" },
              },
            },
          },
        },
      },
    },
  },
};

module.exports = openApiSpec;
