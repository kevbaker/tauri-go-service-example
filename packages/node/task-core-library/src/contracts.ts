import type {
  CreateTaskInput,
  ListTasksQuery,
  Task,
  TaskBridgeRequest,
  TaskStatus,
  UpdateTaskInput,
} from "./generated/task-api";

export type {
  ApplicationError,
  CreateTaskInput,
  CreateTaskRequest,
  DeleteTaskRequest,
  FailedTaskResponse,
  GetTaskRequest,
  ListTasksQuery,
  ListTasksRequest,
  SuccessfulTaskResponse,
  Task,
  TaskBridgeResponse,
  TaskBridgeRequest,
  TaskClientErrorCode,
  TaskIdPayload,
  TaskStatus,
  UpdateTaskInput,
  UpdateTaskPayload,
  UpdateTaskRequest,
} from "./generated/task-api";

export type TaskOperation = TaskBridgeRequest["operation"];

export const taskStatuses = [
  "todo",
  "in_progress",
  "done",
] as const satisfies readonly TaskStatus[];

export const taskOperations = [
  "tasks.list",
  "tasks.get",
  "tasks.create",
  "tasks.update",
  "tasks.delete",
] as const satisfies readonly TaskOperation[];

export const taskProtocolVersion = 1 as const;

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
