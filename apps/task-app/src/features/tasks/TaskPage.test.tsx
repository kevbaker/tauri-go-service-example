import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import {
  TaskClientError,
  type Task,
  type TaskCoreClient,
} from "@tauri-go-service-example/task-core-library";
import { TaskPage } from "./TaskPage";

vi.mock("@vaadin/react-components", () => ({
  Button: ({
    children,
    disabled,
    onClick,
  }: {
    children: ReactNode;
    disabled?: boolean;
    onClick?(): void;
    theme?: string;
  }) => (
    <button type="button" disabled={disabled} onClick={onClick}>
      {children}
    </button>
  ),
  TextField: ({
    errorMessage,
    invalid,
    label,
    maxlength,
    onValueChanged,
    required,
    value,
  }: {
    errorMessage?: string;
    invalid?: boolean;
    label: string;
    maxlength?: number;
    onValueChanged(event: { detail: { value: string } }): void;
    required?: boolean;
    value: string;
  }) => (
    <label>
      {label}
      <input
        aria-invalid={invalid}
        maxLength={maxlength}
        required={required}
        value={value}
        onChange={(event) => onValueChanged({ detail: { value: event.target.value } })}
      />
      {invalid ? <span>{errorMessage}</span> : null}
    </label>
  ),
  TextArea: ({
    label,
    maxlength,
    onValueChanged,
    value,
  }: {
    className?: string;
    label: string;
    maxlength?: number;
    onValueChanged(event: { detail: { value: string } }): void;
    value: string;
  }) => (
    <label>
      {label}
      <textarea
        maxLength={maxlength}
        value={value}
        onChange={(event) => onValueChanged({ detail: { value: event.target.value } })}
      />
    </label>
  ),
  Select: ({
    disabled,
    items,
    label,
    onValueChanged,
    value,
  }: {
    disabled?: boolean;
    items: { label: string; value: string }[];
    label: string;
    onValueChanged(event: { detail: { value: string } }): void;
    value: string;
  }) => (
    <label>
      {label}
      <select
        disabled={disabled}
        value={value}
        onChange={(event) => onValueChanged({ detail: { value: event.target.value } })}
      >
        {items.map((item) => (
          <option key={item.value} value={item.value}>
            {item.label}
          </option>
        ))}
      </select>
    </label>
  ),
}));

const task: Task = {
  id: "task-1",
  title: "Test the task page",
  description: "Exercise the UI through its typed client.",
  status: "todo",
  createdAt: "2026-09-09T12:00:00.000Z",
  updatedAt: "2026-09-09T12:00:00.000Z",
};

function fakeClient(
  overrides: Partial<TaskCoreClient["tasks"]> = {},
): TaskCoreClient {
  return {
    tasks: {
      list: vi.fn().mockResolvedValue([task]),
      get: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      ...overrides,
    },
  };
}

describe("TaskPage", () => {
  it("loads tasks and reports the summary", async () => {
    render(<TaskPage client={fakeClient()} />);

    expect(screen.getByText("Loading tasks…")).toBeInTheDocument();
    expect(await screen.findByText(task.title)).toBeInTheDocument();

    const summary = screen.getByRole("region", { name: "Task summary" });
    expect(within(summary).getByText("Total").previousElementSibling).toHaveTextContent("1");
    expect(within(summary).getByText("Open").previousElementSibling).toHaveTextContent("1");
    expect(within(summary).getByText("Done").previousElementSibling).toHaveTextContent("0");
  });

  it("creates a task with values from the form", async () => {
    const created = { ...task, id: "task-2", title: "New task", status: "done" as const };
    const client = fakeClient({ create: vi.fn().mockResolvedValue(created) });
    render(<TaskPage client={client} />);
    await screen.findByText(task.title);

    await userEvent.type(screen.getByLabelText("Title"), "New task");
    const [createStatus] = screen.getAllByLabelText("Status");
    if (!createStatus) throw new Error("Create status control was not rendered");
    await userEvent.selectOptions(createStatus, "done");
    await userEvent.type(
      screen.getByLabelText("Description"),
      "Created in a component test",
    );
    await userEvent.click(screen.getByText("Add task"));

    await waitFor(() =>
      expect(client.tasks.create).toHaveBeenCalledWith({
        title: "New task",
        description: "Created in a component test",
        status: "done",
      }),
    );
    expect(screen.getByRole("heading", { level: 3, name: "New task" })).toBeInTheDocument();
  });

  it("updates status and deletes only after confirmation", async () => {
    const updated = { ...task, status: "done" as const };
    const client = fakeClient({
      update: vi.fn().mockResolvedValue(updated),
      delete: vi.fn().mockResolvedValue(undefined),
    });
    const confirm = vi.spyOn(window, "confirm");
    const { container } = render(<TaskPage client={client} />);
    await screen.findByText(task.title);

    const card = container.querySelector(".task-card")!;
    await userEvent.selectOptions(within(card as HTMLElement).getByLabelText("Status"), "done");
    await waitFor(() =>
      expect(client.tasks.update).toHaveBeenCalledWith(task.id, { status: "done" }),
    );

    confirm.mockReturnValueOnce(false);
    await userEvent.click(within(card as HTMLElement).getByText("Delete"));
    expect(client.tasks.delete).not.toHaveBeenCalled();

    confirm.mockReturnValueOnce(true);
    await userEvent.click(within(card as HTMLElement).getByText("Delete"));
    await waitFor(() => expect(client.tasks.delete).toHaveBeenCalledWith(task.id));
    expect(screen.queryByText(task.title)).not.toBeInTheDocument();
  });

  it("shows client errors and allows dismissing them", async () => {
    const client = fakeClient({
      list: vi.fn().mockRejectedValue(new TaskClientError("INTERNAL", "Service unavailable.")),
    });
    render(<TaskPage client={client} />);

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Service unavailable.");
    await userEvent.click(within(alert).getByText("Dismiss"));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("uses a safe message for unexpected failures", async () => {
    render(<TaskPage client={fakeClient({ list: vi.fn().mockRejectedValue(new Error("database path")) })} />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Something went wrong. Try again.",
    );
    expect(screen.queryByText("database path")).not.toBeInTheDocument();
  });
});
