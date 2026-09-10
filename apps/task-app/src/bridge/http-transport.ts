import type {
  TaskBackendTransport,
  TaskBridgeRequest,
} from "@tauri-go-service-example/task-core-library";

export interface HttpTaskTransportOptions {
  endpoint?: string;
  fetch?: typeof globalThis.fetch;
}

/** Browser-development adapter. Vite proxies this relative endpoint to Go. */
export class HttpTaskTransport implements TaskBackendTransport {
  private readonly endpoint: string;
  private readonly fetchRequest: typeof globalThis.fetch;

  constructor(options: HttpTaskTransportOptions = {}) {
    this.endpoint = options.endpoint ?? "/task-service/v1/invoke";
    this.fetchRequest = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  async invoke(request: TaskBridgeRequest): Promise<unknown> {
    const response = await this.fetchRequest(this.endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new Error(`Task service HTTP status ${response.status.toString()}`);
    }

    return response.json();
  }
}
