import {
  taskProtocolVersion,
  taskStatuses,
  type CreateTaskInput,
  type ListTasksQuery,
  type Task,
  type TaskBackendTransport,
  type TaskBridgeRequest,
  type TaskCoreClient,
  type TaskStatus,
  type UpdateTaskInput,
} from "./contracts";
import {
  TaskClientError,
  type TaskClientErrorCode,
} from "./errors";

export interface CreateTaskCoreClientOptions {
  createRequestId?: () => string;
}

let fallbackRequestSequence = 0;

function defaultRequestId(): string {
  if (typeof globalThis.crypto?.randomUUID === "function") {
    return globalThis.crypto.randomUUID();
  }

  fallbackRequestSequence += 1;
  return `request-${Date.now().toString(36)}-${fallbackRequestSequence.toString(36)}`;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function protocolError(message: string, requestId?: string): TaskClientError {
  return new TaskClientError(
    "PROTOCOL_ERROR",
    message,
    requestId === undefined ? {} : { requestId },
  );
}

function isTaskStatus(value: unknown): value is TaskStatus {
  return (
    typeof value === "string" &&
    taskStatuses.some((status) => status === value)
  );
}

function parseTask(value: unknown, requestId: string): Task {
  if (!isRecord(value)) {
    throw protocolError("The task service returned an invalid task.", requestId);
  }

  const { id, title, description, status, createdAt, updatedAt } = value;
  if (
    typeof id !== "string" ||
    typeof title !== "string" ||
    (description !== null && typeof description !== "string") ||
    !isTaskStatus(status) ||
    typeof createdAt !== "string" ||
    typeof updatedAt !== "string"
  ) {
    throw protocolError("The task service returned an invalid task.", requestId);
  }

  return { id, title, description, status, createdAt, updatedAt };
}

function parseTaskList(value: unknown, requestId: string): Task[] {
  if (!Array.isArray(value)) {
    throw protocolError("The task service returned an invalid task list.", requestId);
  }
  return value.map((item) => parseTask(item, requestId));
}

function parseFieldErrors(value: unknown): Readonly<Record<string, string>> | undefined {
  if (!isRecord(value)) return undefined;

  const entries = Object.entries(value);
  if (!entries.every((entry) => typeof entry[1] === "string")) return undefined;
  return Object.fromEntries(entries) as Record<string, string>;
}

function parseErrorCode(value: unknown): TaskClientErrorCode {
  switch (value) {
    case "VALIDATION":
    case "NOT_FOUND":
    case "CONFLICT":
    case "INTERNAL":
    case "SERVICE_UNAVAILABLE":
      return value;
    default:
      return "PROTOCOL_ERROR";
  }
}

function unwrapResponse<T>(
  value: unknown,
  requestId: string,
  parseData: (data: unknown, requestId: string) => T,
): T {
  if (!isRecord(value)) {
    throw protocolError("The task service returned an invalid response.", requestId);
  }
  if (
    value.protocolVersion !== taskProtocolVersion ||
    value.requestId !== requestId ||
    typeof value.ok !== "boolean"
  ) {
    throw protocolError("The task service response did not match the request.", requestId);
  }

  if (!value.ok) {
    if (!isRecord(value.error) || typeof value.error.message !== "string") {
      throw protocolError("The task service returned an invalid error.", requestId);
    }
    const fieldErrors = parseFieldErrors(value.error.fieldErrors);
    throw new TaskClientError(parseErrorCode(value.error.code), value.error.message, {
      requestId,
      ...(fieldErrors === undefined ? {} : { fieldErrors }),
    });
  }

  return parseData(value.data, requestId);
}

export function createTaskCoreClient(
  transport: TaskBackendTransport,
  options: CreateTaskCoreClientOptions = {},
): TaskCoreClient {
  const createRequestId = options.createRequestId ?? defaultRequestId;

  async function invoke<T>(
    request: TaskBridgeRequest,
    parseData: (data: unknown, requestId: string) => T,
  ): Promise<T> {
    const { requestId } = request;
    let response: unknown;
    try {
      response = await transport.invoke(request);
    } catch (error) {
      if (error instanceof TaskClientError) throw error;
      throw new TaskClientError(
        "SERVICE_UNAVAILABLE",
        "The task service is unavailable.",
        { requestId },
      );
    }
    return unwrapResponse(response, requestId, parseData);
  }

  return {
    tasks: {
      list(query: ListTasksQuery = {}) {
        return invoke(
          {
            protocolVersion: taskProtocolVersion,
            requestId: createRequestId(),
            operation: "tasks.list",
            payload: query,
          },
          parseTaskList,
        );
      },
      get(id: string) {
        return invoke(
          {
            protocolVersion: taskProtocolVersion,
            requestId: createRequestId(),
            operation: "tasks.get",
            payload: { id },
          },
          parseTask,
        );
      },
      create(input: CreateTaskInput) {
        return invoke(
          {
            protocolVersion: taskProtocolVersion,
            requestId: createRequestId(),
            operation: "tasks.create",
            payload: input,
          },
          parseTask,
        );
      },
      update(id: string, input: UpdateTaskInput) {
        return invoke(
          {
            protocolVersion: taskProtocolVersion,
            requestId: createRequestId(),
            operation: "tasks.update",
            payload: { id, input },
          },
          parseTask,
        );
      },
      async delete(id: string) {
        await invoke(
          {
            protocolVersion: taskProtocolVersion,
            requestId: createRequestId(),
            operation: "tasks.delete",
            payload: { id },
          },
          () => undefined,
        );
      },
    },
  };
}
