import { describe, expect, it, vi } from "vitest";

import {
  createTaskCoreClient,
  TaskClientError,
  taskProtocolVersion,
  type Task,
  type TaskBackendTransport,
} from "./index";

const task: Task = {
  id: "task-1",
  title: "Use the core library",
  description: null,
  status: "todo",
  createdAt: "2026-09-10T00:00:00Z",
  updatedAt: "2026-09-10T00:00:00Z",
};

describe("createTaskCoreClient", () => {
  it("maps the five CRUD methods to the versioned backend operations", async () => {
    const invoke = vi.fn<TaskBackendTransport["invoke"]>(async (request) => ({
      protocolVersion: taskProtocolVersion,
      requestId: request.requestId,
      ok: true,
      data: request.operation === "tasks.list" ? [task] : task,
    }));
    const client = createTaskCoreClient(
      { invoke },
      { createRequestId: () => "request-1" },
    );

    await expect(client.tasks.list({ status: "todo" })).resolves.toEqual([task]);
    await expect(client.tasks.get("task-1")).resolves.toEqual(task);
    await expect(client.tasks.create({ title: "Create" })).resolves.toEqual(task);
    await expect(
      client.tasks.update("task-1", { description: null }),
    ).resolves.toEqual(task);
    await expect(client.tasks.delete("task-1")).resolves.toBeUndefined();

    expect(invoke.mock.calls.map(([request]) => request)).toEqual([
      {
        protocolVersion: 1,
        requestId: "request-1",
        operation: "tasks.list",
        payload: { status: "todo" },
      },
      {
        protocolVersion: 1,
        requestId: "request-1",
        operation: "tasks.get",
        payload: { id: "task-1" },
      },
      {
        protocolVersion: 1,
        requestId: "request-1",
        operation: "tasks.create",
        payload: { title: "Create" },
      },
      {
        protocolVersion: 1,
        requestId: "request-1",
        operation: "tasks.update",
        payload: { id: "task-1", input: { description: null } },
      },
      {
        protocolVersion: 1,
        requestId: "request-1",
        operation: "tasks.delete",
        payload: { id: "task-1" },
      },
    ]);
  });

  it("normalizes application errors and retains correlation details", async () => {
    const client = createTaskCoreClient(
      {
        async invoke(request) {
          return {
            protocolVersion: 1,
            requestId: request.requestId,
            ok: false,
            error: {
              code: "VALIDATION",
              message: "The request is invalid",
              fieldErrors: { title: "Title is required" },
            },
          };
        },
      },
      { createRequestId: () => "request-validation" },
    );

    await expect(client.tasks.create({ title: "" })).rejects.toEqual(
      new TaskClientError("VALIDATION", "The request is invalid", {
        requestId: "request-validation",
        fieldErrors: { title: "Title is required" },
      }),
    );
  });

  it("rejects malformed and mismatched backend responses", async () => {
    const client = createTaskCoreClient(
      {
        async invoke() {
          return { protocolVersion: 1, requestId: "another-request", ok: true };
        },
      },
      { createRequestId: () => "request-1" },
    );

    await expect(client.tasks.list()).rejects.toMatchObject({
      name: "TaskClientError",
      code: "PROTOCOL_ERROR",
      requestId: "request-1",
    });
  });

  it("does not expose raw transport failures", async () => {
    const client = createTaskCoreClient(
      {
        async invoke() {
          throw new Error("connection details");
        },
      },
      { createRequestId: () => "request-offline" },
    );

    await expect(client.tasks.list()).rejects.toMatchObject({
      name: "TaskClientError",
      code: "SERVICE_UNAVAILABLE",
      message: "The task service is unavailable.",
      requestId: "request-offline",
    });
  });
});
