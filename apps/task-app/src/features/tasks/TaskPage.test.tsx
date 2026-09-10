import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ButtonHTMLAttributes, ReactNode } from "react";
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
    theme: _theme,
    ...props
  }: {
    children: ReactNode;
    theme?: string;
  } & ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
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
  Dialog: ({
    children,
    opened,
    "aria-label": ariaLabel,
  }: {
    children?: ReactNode;
    opened?: boolean;
    "aria-label"?: string;
    noCloseOnEsc?: boolean;
    noCloseOnOutsideClick?: boolean;
    onClosed?(): void;
    theme?: string;
  }) =>
    opened ? (
      <div role="dialog" aria-label={ariaLabel}>
        {children}
      </div>
    ) : null,
  MenuBar: ({
    items,
    "aria-label": ariaLabel,
  }: {
    items?: {
      component?: ReactNode;
      text?: string;
      children?: { text?: string }[];
    }[];
    "aria-label"?: string;
    theme?: string;
  }) => (
    <nav aria-label={ariaLabel}>
      {items?.map((item) => (
        <div key={item.text ?? "application-menu"}>
          <button type="button">{item.component ?? item.text}</button>
          {item.children?.map((child) => (
            <span key={child.text}>{child.text}</span>
          ))}
        </div>
      ))}
    </nav>
  ),
  Icon: ({ icon }: { icon?: string }) => <span aria-hidden="true">{icon}</span>,
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
    config: {
      getPublic: vi.fn().mockResolvedValue({
        environment: "test",
        ui: { appName: "Tasks", pageSize: 25, refreshIntervalMs: 30_000 },
      }),
    },
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

  it("refreshes on demand and polls for backend task changes", async () => {
    const changedTask = { ...task, id: "task-2", title: "Created elsewhere" };
    const list = vi
      .fn()
      .mockResolvedValueOnce([task])
      .mockResolvedValue([changedTask, task]);
    const client = fakeClient({ list });
    let poll: (() => void) | undefined;
    const setInterval = vi
      .spyOn(window, "setInterval")
      .mockImplementation((handler: TimerHandler, timeout?: number) => {
        if (timeout === 5_000) poll = handler as () => void;
        return 1;
      });
    const clearInterval = vi.spyOn(window, "clearInterval").mockImplementation(() => {});

    const { unmount } = render(
      <TaskPage client={client} refreshIntervalMs={5_000} />,
    );
    await screen.findByText(task.title);

    await userEvent.click(screen.getByText("Refresh"));
    expect(await screen.findByText(changedTask.title)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByText("Refresh")).toBeEnabled());

    const runPoll = poll;
    if (!runPoll) throw new Error("Task polling interval was not registered");
    await act(async () => {
      runPoll();
    });
    await waitFor(() => expect(list).toHaveBeenCalledTimes(3));

    unmount();
    expect(clearInterval).toHaveBeenCalledWith(1);
    setInterval.mockRestore();
    clearInterval.mockRestore();
  });

  it("creates a task with values from the form", async () => {
    const created = { ...task, id: "task-2", title: "New task" };
    const client = fakeClient({ create: vi.fn().mockResolvedValue(created) });
    render(<TaskPage client={client} />);
    await screen.findByText(task.title);

    expect(screen.queryByLabelText("Title")).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Create task" }));
    const dialog = screen.getByRole("dialog", { name: "Create task" });

    await userEvent.type(within(dialog).getByLabelText("Title"), "New task");
    expect(within(dialog).queryByLabelText("Status")).not.toBeInTheDocument();
    expect(within(dialog).queryByLabelText("Description")).not.toBeInTheDocument();
    await userEvent.click(
      within(dialog).getByRole("button", { name: "Create task" }),
    );

    await waitFor(() =>
      expect(client.tasks.create).toHaveBeenCalledWith({
        title: "New task",
        description: "",
        status: "todo",
      }),
    );
    expect(screen.getByRole("heading", { level: 3, name: "New task" })).toBeInTheDocument();
    expect(screen.queryByRole("dialog", { name: "Create task" })).not.toBeInTheDocument();
  });

  it("opens the same task sheet for editing", async () => {
    render(<TaskPage client={fakeClient()} />);
    await screen.findByText(task.title);

    await userEvent.click(
      screen.getByRole("button", { name: `Edit ${task.title}` }),
    );
    const dialog = screen.getByRole("dialog", { name: "Edit task" });

    expect(within(dialog).getByLabelText("Title")).toHaveValue(task.title);
    expect(within(dialog).getByLabelText("Status")).toHaveValue("todo");
    expect(within(dialog).getByLabelText("Description")).toHaveValue(
      task.description,
    );
    expect(
      within(dialog).getByRole("button", { name: "Save changes" }),
    ).toBeInTheDocument();
  });

  it("provides an application menu with an About item", async () => {
    render(<TaskPage client={fakeClient()} />);
    await screen.findByText(task.title);

    expect(screen.getByRole("button", { name: "Menu" })).toBeInTheDocument();
    expect(screen.getByText("About")).toBeInTheDocument();
    expect(screen.getByText("Release notes")).toBeInTheDocument();
    expect(screen.getByText("Help")).toBeInTheDocument();
  });

  it("updates status and deletes only after confirmation", async () => {
    const inProgress = { ...task, status: "in_progress" as const };
    const done = { ...task, status: "done" as const };
    const pendingAgain = { ...task, status: "todo" as const };
    const client = fakeClient({
      update: vi
        .fn()
        .mockResolvedValueOnce(inProgress)
        .mockResolvedValueOnce(done)
        .mockResolvedValueOnce(pendingAgain),
      delete: vi.fn().mockResolvedValue(undefined),
    });
    const confirm = vi.spyOn(window, "confirm");
    const { container } = render(<TaskPage client={client} />);
    await screen.findByText(task.title);

    const card = container.querySelector(".task-card")!;
    const description = task.description;
    if (!description) throw new Error("Task fixture requires a description");
    expect(within(card as HTMLElement).queryByText(description)).not.toBeInTheDocument();
    expect(within(card as HTMLElement).getByTitle(description)).toBeInTheDocument();
    expect(within(card as HTMLElement).queryByLabelText("Status")).not.toBeInTheDocument();
    await userEvent.click(
      within(card as HTMLElement).getByRole("button", {
        name: "Status: Pending. Change to In progress",
      }),
    );
    await waitFor(() =>
      expect(client.tasks.update).toHaveBeenCalledWith(task.id, { status: "in_progress" }),
    );
    await userEvent.click(
      within(card as HTMLElement).getByRole("button", {
        name: "Status: In progress. Change to Done",
      }),
    );
    await waitFor(() =>
      expect(client.tasks.update).toHaveBeenLastCalledWith(task.id, { status: "done" }),
    );
    await userEvent.click(
      within(card as HTMLElement).getByRole("button", {
        name: "Status: Done. Change to Pending",
      }),
    );
    await waitFor(() =>
      expect(client.tasks.update).toHaveBeenLastCalledWith(task.id, { status: "todo" }),
    );

    confirm.mockReturnValueOnce(false);
    await userEvent.click(
      within(card as HTMLElement).getByRole("button", { name: `Delete ${task.title}` }),
    );
    expect(client.tasks.delete).not.toHaveBeenCalled();

    confirm.mockReturnValueOnce(true);
    await userEvent.click(
      within(card as HTMLElement).getByRole("button", { name: `Delete ${task.title}` }),
    );
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
