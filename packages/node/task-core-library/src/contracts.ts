export const taskStatuses = ["todo", "in_progress", "done"] as const;

export type TaskStatus = (typeof taskStatuses)[number];

export interface Task {
  id: string;
  title: string;
  description: string | null;
  status: TaskStatus;
  createdAt: string;
  updatedAt: string;
}

export interface ListTasksQuery {
  status?: TaskStatus;
  limit?: number;
  offset?: number;
}

export interface CreateTaskInput {
  title: string;
  description?: string | null;
  status?: TaskStatus;
}

export interface UpdateTaskInput {
  title?: string;
  description?: string | null;
  status?: TaskStatus;
}

export const taskOperations = [
  "tasks.list",
  "tasks.get",
  "tasks.create",
  "tasks.update",
  "tasks.delete",
] as const;

export type TaskOperation = (typeof taskOperations)[number];

export const taskProtocolVersion = 1 as const;

export interface TaskBridgeRequest {
  protocolVersion: typeof taskProtocolVersion;
  requestId: string;
  operation: TaskOperation;
  payload: unknown;
}

/**
 * The application shell supplies this transport. It may use Tauri IPC, HTTP,
 * or an in-process test adapter, but it must return the shared bridge envelope.
 */
export interface TaskBackendTransport {
  invoke(request: TaskBridgeRequest): Promise<unknown>;
}

export interface TaskCrudClient {
  list(query?: ListTasksQuery): Promise<Task[]>;
  get(id: string): Promise<Task>;
  create(input: CreateTaskInput): Promise<Task>;
  update(id: string, input: UpdateTaskInput): Promise<Task>;
  delete(id: string): Promise<void>;
}

export interface TaskCoreClient {
  readonly tasks: TaskCrudClient;
}
