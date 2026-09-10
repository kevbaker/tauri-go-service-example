import { afterEach, describe, expect, it, vi } from "vitest";

import { createInMemoryAppClient } from "./in-memory-app-client";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("createInMemoryAppClient", () => {
  it("supports create, update, list, and delete without exposing mutable state", async () => {
    const client = createInMemoryAppClient();
    const initial = await client.tasks.list();

    const created = await client.tasks.create({
      title: "  Test the complete flow  ",
      description: "  Through the typed client  ",
      status: "todo",
    });

    expect(created).toMatchObject({
      title: "Test the complete flow",
      description: "Through the typed client",
      status: "todo",
    });
    expect(await client.tasks.list()).toHaveLength(initial.length + 1);
    await expect(client.tasks.get(created.id)).resolves.toEqual(created);

    created.title = "Changed outside the client";
    const listed = await client.tasks.list();
    expect(listed[0]?.title).toBe("Test the complete flow");

    const updated = await client.tasks.update(created.id, {
      description: "  Updated description  ",
      status: "done",
    });
    expect(updated).toMatchObject({
      title: "Test the complete flow",
      description: "Updated description",
      status: "done",
    });

    await client.tasks.delete(created.id);
    expect(await client.tasks.list()).toHaveLength(initial.length);
  });

  it.each(["", "   "])("rejects a blank title (%j)", async (title) => {
    const client = createInMemoryAppClient();

    await expect(
      client.tasks.create({ title, description: "", status: "todo" }),
    ).rejects.toEqual(
      expect.objectContaining({
        code: "VALIDATION",
        message: "Enter a task title.",
        name: "TaskClientError",
      }),
    );
  });

  it("rejects titles longer than the UI contract permits", async () => {
    const client = createInMemoryAppClient();

    await expect(
      client.tasks.create({
        title: "x".repeat(121),
        description: "",
        status: "todo",
      }),
    ).rejects.toMatchObject({ code: "VALIDATION" });
  });

  it("returns NOT_FOUND for mutations of missing tasks", async () => {
    const client = createInMemoryAppClient();

    await expect(
      client.tasks.update("missing", { status: "done" }),
    ).rejects.toMatchObject({ code: "NOT_FOUND" });
    await expect(client.tasks.get("missing")).rejects.toMatchObject({
      code: "NOT_FOUND",
    });
    await expect(client.tasks.delete("missing")).rejects.toMatchObject({
      code: "NOT_FOUND",
    });
  });

  it("creates unique IDs when randomUUID is unavailable on a LAN HTTP origin", async () => {
    vi.stubGlobal("crypto", {});
    const client = createInMemoryAppClient();

    const first = await client.tasks.create({
      title: "First LAN task",
      description: "",
      status: "todo",
    });
    const second = await client.tasks.create({
      title: "Second LAN task",
      description: "",
      status: "todo",
    });

    expect(first.id).toMatch(/^task-/);
    expect(second.id).toMatch(/^task-/);
    expect(second.id).not.toBe(first.id);
  });
});
