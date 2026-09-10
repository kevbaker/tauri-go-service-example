import {
  createTaskCoreClient,
  type TaskCoreClient,
} from "@tauri-go-service-example/task-core-library";
import { DesktopTaskTransport } from "./desktop-transport";
import { HttpTaskTransport } from "./http-transport";

export interface TaskAppRuntime {
  client: TaskCoreClient;
  backendLabel: string;
}

export function isTauriRuntime(): boolean {
  return "__TAURI_INTERNALS__" in globalThis;
}

export function createTaskAppRuntime(): TaskAppRuntime {
  if (isTauriRuntime()) {
    return {
      client: createTaskCoreClient(new DesktopTaskTransport()),
      backendLabel: "Go + SQLite · desktop IPC",
    };
  }

  return {
    client: createTaskCoreClient(new HttpTaskTransport()),
    backendLabel: "Go + SQLite · browser HTTP",
  };
}
