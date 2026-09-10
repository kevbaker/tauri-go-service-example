import { describe, expect, it, vi } from "vitest";
import { taskProtocolVersion, type TaskBridgeRequest } from "@tauri-go-service-example/task-core-library";
import { DesktopTaskTransport } from "./desktop-transport";
import { HttpTaskTransport } from "./http-transport";

const request: TaskBridgeRequest = {
  protocolVersion: taskProtocolVersion,
  requestId: "transport-test",
  operation: "tasks.list",
  payload: {},
};

describe("task transports", () => {
  it("maps the desktop transport to one capability-scoped Tauri command", async () => {
    const invoke = vi.fn().mockResolvedValue({ ok: true });
    const transport = new DesktopTaskTransport(invoke);

    await expect(transport.invoke(request)).resolves.toEqual({ ok: true });
    expect(invoke).toHaveBeenCalledWith("service_invoke", { request });
  });

  it("posts the same envelope through the browser development endpoint", async () => {
    const fetchRequest = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    const transport = new HttpTaskTransport({ fetch: fetchRequest });

    await expect(transport.invoke(request)).resolves.toEqual({ ok: true });
    expect(fetchRequest).toHaveBeenCalledWith("/task-service/v1/invoke", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
    });
  });

  it("rejects non-success HTTP responses without leaking their bodies", async () => {
    const transport = new HttpTaskTransport({
      fetch: vi.fn().mockResolvedValue(new Response("database detail", { status: 503 })),
    });

    await expect(transport.invoke(request)).rejects.toThrow("HTTP status 503");
  });
});
