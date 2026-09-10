/* This file is generated from contracts/task-api.schema.json. Do not edit. */

/**
 * Canonical JSON wire contract for task requests and responses.
 */
export type TaskAPIContract = TaskBridgeRequest | TaskBridgeResponse;
export type TaskBridgeRequest =
  | GetPublicConfigRequest
  | ListTasksRequest
  | GetTaskRequest
  | CreateTaskRequest
  | UpdateTaskRequest
  | DeleteTaskRequest;
export type TaskStatus = "todo" | "in_progress" | "done";
export type TaskBridgeResponse = SuccessfulTaskResponse | FailedTaskResponse;
export type TaskClientErrorCode =
  "VALIDATION" | "NOT_FOUND" | "CONFLICT" | "INTERNAL" | "SERVICE_UNAVAILABLE" | "PROTOCOL_ERROR";

export interface GetPublicConfigRequest {
  protocolVersion: 1;
  requestId: string;
  operation: "config.getPublic";
  payload: {};
}
export interface ListTasksRequest {
  protocolVersion: 1;
  requestId: string;
  operation: "tasks.list";
  payload: ListTasksQuery;
}
export interface ListTasksQuery {
  status?: TaskStatus;
  limit?: number;
  offset?: number;
}
export interface GetTaskRequest {
  protocolVersion: 1;
  requestId: string;
  operation: "tasks.get";
  payload: TaskIdPayload;
}
export interface TaskIdPayload {
  id: string;
}
export interface CreateTaskRequest {
  protocolVersion: 1;
  requestId: string;
  operation: "tasks.create";
  payload: CreateTaskInput;
}
export interface CreateTaskInput {
  title: string;
  description?: string | null;
  status?: TaskStatus;
}
export interface UpdateTaskRequest {
  protocolVersion: 1;
  requestId: string;
  operation: "tasks.update";
  payload: UpdateTaskPayload;
}
export interface UpdateTaskPayload {
  id: string;
  input: UpdateTaskInput;
}
export interface UpdateTaskInput {
  title?: string;
  description?: string | null;
  status?: TaskStatus;
}
export interface DeleteTaskRequest {
  protocolVersion: 1;
  requestId: string;
  operation: "tasks.delete";
  payload: TaskIdPayload;
}
export interface SuccessfulTaskResponse {
  protocolVersion: 1;
  requestId: string;
  ok: true;
  data: PublicConfig | Task | Task[] | {};
}
export interface PublicConfig {
  environment: string;
  ui: {
    appName: string;
    pageSize: number;
    refreshIntervalMs: number;
  };
}
export interface Task {
  id: string;
  title: string;
  description: string | null;
  status: TaskStatus;
  createdAt: string;
  updatedAt: string;
}
export interface FailedTaskResponse {
  protocolVersion: 1;
  requestId: string;
  ok: false;
  error: ApplicationError;
}
export interface ApplicationError {
  code: TaskClientErrorCode;
  message: string;
  fieldErrors?: {
    [k: string]: string;
  };
}
