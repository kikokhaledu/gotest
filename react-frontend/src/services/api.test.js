import { describe, expect, it } from "vitest";
import * as api from "./api";

describe("api service exports", () => {
	it("exposes all expected API functions", () => {
		const expectedFunctions = [
			"checkHealth",
			"getUsers",
			"getUserById",
			"getTasks",
			"getStats",
			"getTaskHistory",
			"createUser",
			"createTask",
			"updateTask",
		];

		for (const fnName of expectedFunctions) {
			expect(typeof api[fnName]).toBe("function");
		}
	});
});
