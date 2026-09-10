import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  Button,
  Dialog,
  Icon,
  MenuBar,
  type MenuBarItem,
} from "@vaadin/react-components";
import {
  TaskClientError,
  type CreateTaskInput,
  type Task,
  type TaskCoreClient,
  type TaskStatus,
} from "@tauri-go-service-example/task-core-library";
import { TaskCard } from "./TaskCard";
import { TaskForm } from "./TaskForm";
import "./tasks.css";

interface TaskPageProps {
  client: TaskCoreClient;
  backendLabel?: string;
  refreshIntervalMs?: number;
}

const DEFAULT_REFRESH_INTERVAL_MS = 30_000;
const applicationMenuItems: MenuBarItem[] = [
  {
    component: (
      <span className="application-menu__trigger">
        <Icon icon="vaadin:menu" aria-hidden="true" />
        <span className="visually-hidden">Menu</span>
      </span>
    ),
    children: [
      { text: "About" },
      { text: "Release notes" },
      { text: "Help" },
    ],
  },
];

function errorMessage(error: unknown): string {
  if (error instanceof TaskClientError) {
    return error.message;
  }

  return "Something went wrong. Try again.";
}

export function TaskPage({
  client,
  backendLabel = "Injected task client",
  refreshIntervalMs = DEFAULT_REFRESH_INTERVAL_MS,
}: TaskPageProps) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const [taskFormOpened, setTaskFormOpened] = useState(false);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [busyTaskId, setBusyTaskId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const active = useRef(false);
  const listInFlight = useRef(false);
  const dataVersion = useRef(0);

  const doneCount = useMemo(
    () => tasks.filter((task) => task.status === "done").length,
    [tasks],
  );

  const refreshTasks = useCallback(
    async (initial = false) => {
      if (listInFlight.current) return;
      listInFlight.current = true;
      const versionAtStart = dataVersion.current;
      if (initial) setLoading(true);
      else setRefreshing(true);

      try {
        const items = await client.tasks.list();
        if (active.current && dataVersion.current === versionAtStart) {
          setTasks(items);
        }
      } catch (reason) {
        if (active.current) setError(errorMessage(reason));
      } finally {
        listInFlight.current = false;
        if (active.current) {
          setLoading(false);
          setRefreshing(false);
        }
      }
    },
    [client],
  );

  useEffect(() => {
    active.current = true;
    void refreshTasks(true);
    const interval = window.setInterval(() => {
      void refreshTasks();
    }, refreshIntervalMs);

    return () => {
      active.current = false;
      window.clearInterval(interval);
    };
  }, [refreshIntervalMs, refreshTasks]);

  async function handleSubmit(input: CreateTaskInput) {
    dataVersion.current += 1;
    setSaving(true);
    setError("");

    try {
      if (editingTask) {
        const updated = await client.tasks.update(editingTask.id, input);
        dataVersion.current += 1;
        setTasks((current) =>
          current.map((task) => (task.id === updated.id ? updated : task)),
        );
      } else {
        const created = await client.tasks.create(input);
        dataVersion.current += 1;
        setTasks((current) => [
          created,
          ...current.filter((task) => task.id !== created.id),
        ]);
      }
      setTaskFormOpened(false);
      setEditingTask(null);
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setSaving(false);
    }
  }

  async function handleStatusChange(task: Task, status: TaskStatus) {
    if (task.status === status) return;

    dataVersion.current += 1;
    setBusyTaskId(task.id);
    setError("");
    try {
      const updated = await client.tasks.update(task.id, { status });
      dataVersion.current += 1;
      setTasks((current) =>
        current.map((item) => (item.id === updated.id ? updated : item)),
      );
      setEditingTask((current) =>
        current?.id === updated.id ? updated : current,
      );
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setBusyTaskId(null);
    }
  }

  async function handleDelete(task: Task) {
    if (!window.confirm(`Delete “${task.title}”?`)) return;

    dataVersion.current += 1;
    setBusyTaskId(task.id);
    setError("");
    try {
      await client.tasks.delete(task.id);
      dataVersion.current += 1;
      setTasks((current) => current.filter((item) => item.id !== task.id));
      if (editingTask?.id === task.id) {
        setTaskFormOpened(false);
        setEditingTask(null);
      }
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setBusyTaskId(null);
    }
  }

  function openCreateTask() {
    setEditingTask(null);
    setTaskFormOpened(true);
  }

  function openEditTask(task: Task) {
    setEditingTask(task);
    setTaskFormOpened(true);
  }

  function closeTaskForm() {
    setTaskFormOpened(false);
    setEditingTask(null);
  }

  return (
    <main className="task-shell">
      <header className="app-header">
        <div>
          <p className="eyebrow">Tauri + Go proof of concept</p>
          <h1>Tasks</h1>
          <p className="app-header__intro">
            A small CRUD surface for proving the typed service bridge.
          </p>
        </div>
        <div className="app-header__actions">
          <MenuBar
            aria-label="Application menu"
            items={applicationMenuItems}
            theme="tertiary"
          />
          <div className="preview-badge" title="Tasks are persisted by the Go service">
            {backendLabel}
          </div>
        </div>
      </header>

      <section className="summary" aria-label="Task summary">
        <div>
          <strong>{tasks.length}</strong>
          <span>Total</span>
        </div>
        <div>
          <strong>{tasks.length - doneCount}</strong>
          <span>Open</span>
        </div>
        <div>
          <strong>{doneCount}</strong>
          <span>Done</span>
        </div>
      </section>

      {error ? (
        <div className="error-banner" role="alert">
          <span>{error}</span>
          <Button theme="tertiary error" onClick={() => setError("")}>
            Dismiss
          </Button>
        </div>
      ) : null}

      <section className="task-list-panel" aria-labelledby="task-list-heading">
        <div className="list-heading">
          <div>
            <p className="eyebrow">Current work</p>
            <h2 id="task-list-heading">Task list</h2>
          </div>
          <div className="list-heading__actions">
            <span>{tasks.length} items</span>
            <Button
              theme="tertiary small"
              disabled={loading || refreshing}
              onClick={() => void refreshTasks()}
            >
              {refreshing ? "Refreshing…" : "Refresh"}
            </Button>
            <Button theme="primary" onClick={openCreateTask}>
              Create task
            </Button>
          </div>
        </div>

        {loading ? (
          <p className="empty-state">Loading tasks…</p>
        ) : tasks.length === 0 ? (
          <div className="empty-state">
            <h3>No tasks yet</h3>
            <p>Create the first task to get started.</p>
          </div>
        ) : (
          <div className="task-list">
            {tasks.map((task) => (
              <TaskCard
                key={task.id}
                task={task}
                busy={busyTaskId === task.id}
                onEdit={openEditTask}
                onDelete={handleDelete}
                onStatusChange={handleStatusChange}
              />
            ))}
          </div>
        )}
      </section>

      <Dialog
        opened={taskFormOpened}
        theme="task-sheet"
        noCloseOnEsc={saving}
        noCloseOnOutsideClick={saving}
        aria-label={editingTask ? "Edit task" : "Create task"}
        onClosed={closeTaskForm}
      >
        {taskFormOpened ? (
          <TaskForm
            editingTask={editingTask}
            busy={saving}
            onCancel={closeTaskForm}
            onSubmit={handleSubmit}
          />
        ) : null}
      </Dialog>
    </main>
  );
}
