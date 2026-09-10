import { invoke } from "@tauri-apps/api/core";
import type {
  TaskBackendTransport,
  TaskBridgeRequest,
} from "@tauri-go-service-example/task-core-library";

export type TauriInvoker = <T>(
  command: string,
  args?: Record<string, unknown>,
) => Promise<T>;

/**
 * Desktop equivalent of an Electron preload adapter. Feature code receives the
 * typed task client and never gets access to Tauri's general invoke primitive.
 */
export class DesktopTaskTransport implements TaskBackendTransport {
  constructor(private readonly invokeCommand: TauriInvoker = invoke) {}

  invoke(request: TaskBridgeRequest): Promise<unknown> {
    return this.invokeCommand("service_invoke", { request });
  }
}
