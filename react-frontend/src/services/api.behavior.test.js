import { beforeEach, describe, expect, it, vi } from "vitest";

const axiosState = vi.hoisted(() => {
	const mockGet = vi.fn();
	const mockPost = vi.fn();
	const mockPut = vi.fn();
	const mockRequest = vi.fn();
	const requestUse = vi.fn();
	const responseUse = vi.fn();
	const mockCreate = vi.fn();
	const mockLoginPost = vi.fn();

	const mockClient = {
		get: mockGet,
		post: mockPost,
		put: mockPut,
		request: mockRequest,
		interceptors: {
			request: {
				use: requestUse,
			},
			response: {
				use: responseUse,
			},
		},
	};

	return {
		mockGet,
		mockPost,
		mockPut,
		mockRequest,
		requestUse,
		responseUse,
		mockCreate,
		mockLoginPost,
		mockClient,
	};
});

vi.mock("axios", () => ({
	default: {
		create: axiosState.mockCreate,
		post: axiosState.mockLoginPost,
	},
}));

beforeEach(() => {
	vi.resetModules();

	axiosState.mockGet.mockReset();
	axiosState.mockPost.mockReset();
	axiosState.mockPut.mockReset();
	axiosState.mockRequest.mockReset();
	axiosState.requestUse.mockReset();
	axiosState.responseUse.mockReset();
	axiosState.mockCreate.mockReset();
	axiosState.mockLoginPost.mockReset();

	axiosState.mockCreate.mockImplementation(() => axiosState.mockClient);
});

describe("api service behavior", () => {
	it("registers request/response interceptors on initialization", async () => {
		await import("./api.js");

		expect(axiosState.mockCreate).toHaveBeenCalledTimes(1);
		expect(axiosState.requestUse).toHaveBeenCalledTimes(1);
		expect(axiosState.responseUse).toHaveBeenCalledTimes(1);
	});

	it("getTasks includes status and user filters when provided", async () => {
		const api = await import("./api.js");
		axiosState.mockGet.mockResolvedValueOnce({ data: { tasks: [] } });

		await api.getTasks("pending", "2");

		expect(axiosState.mockGet).toHaveBeenCalledWith("/api/tasks", {
			params: {
				status: "pending",
				userId: "2",
			},
		});
	});

	it("getTasks sends empty params when filters are omitted", async () => {
		const api = await import("./api.js");
		axiosState.mockGet.mockResolvedValueOnce({ data: { tasks: [] } });

		await api.getTasks();

		expect(axiosState.mockGet).toHaveBeenCalledWith("/api/tasks", {
			params: {},
		});
	});

	it("createUser forwards payload and returns response data", async () => {
		const api = await import("./api.js");
		const payload = {
			name: "Test User",
			email: "test@example.com",
			role: "developer",
		};
		axiosState.mockPost.mockResolvedValueOnce({
			data: {
				id: 10,
				...payload,
			},
		});

		const created = await api.createUser(payload);

		expect(axiosState.mockPost).toHaveBeenCalledWith("/api/users", payload);
		expect(created).toEqual({
			id: 10,
			...payload,
		});
	});

	it("getTaskHistory requests task history endpoint", async () => {
		const api = await import("./api.js");
		axiosState.mockGet.mockResolvedValueOnce({ data: { taskId: 1, history: [] } });

		await api.getTaskHistory(1);

		expect(axiosState.mockGet).toHaveBeenCalledWith("/api/tasks/1/history");
	});

	it("checkHealth wraps upstream errors with context", async () => {
		const api = await import("./api.js");
		axiosState.mockGet.mockRejectedValueOnce(new Error("upstream unavailable"));

		await expect(api.checkHealth()).rejects.toThrow(
			"Health check failed: upstream unavailable"
		);
	});
});
