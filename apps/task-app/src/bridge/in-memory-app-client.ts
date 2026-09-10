import {
  TaskClientError,
  type CreateTaskInput,
  type Task,
  type TaskCoreClient,
  type UpdateTaskInput,
} from "@tauri-go-service-example/task-core-library";

const now = new Date().toISOString();
let fallbackIdSequence = 0;

function createTaskId(): string {
  if (typeof globalThis.crypto?.randomUUID === "function") {
    return globalThis.crypto.randomUUID();
  }

  fallbackIdSequence += 1;
  return `task-${Date.now().toString(36)}-${fallbackIdSequence.toString(36)}`;
}

const starterTasks: Task[] = [
  {
    id: createTaskId(),
    title: "Prove the typed bridge",
    description: "Send the same contract through desktop and browser transports.",
    status: "in_progress",
    createdAt: now,
    updatedAt: now,
  },
  {
    id: createTaskId(),
    title: "Persist tasks in SQLite",
    description: "Connect this UI to the Go service through the typed bridge.",
    status: "todo",
    createdAt: now,
    updatedAt: now,
  },
];

function validateTitle(title: string): string {
  const value = title.trim();
  if (!value) {
    throw new TaskClientError("VALIDATION", "Enter a task title.");
  }

  if (value.length > 120) {
    throw new TaskClientError(
      "VALIDATION",
      "Keep the task title to 120 characters or fewer.",
    );
  }

  return value;
}

function cloneTask(task: Task): Task {
  return { ...task };
}

export function createInMemoryAppClient(): TaskCoreClient {
  let tasks = starterTasks.map(cloneTask);

  return {
    config: {
      async getPublic() {
        return {
          environment: "development",
          ui: {
            appName: "Tauri Go Tasks",
            pageSize: 25,
            refreshIntervalMs: 30_000,
          },
        };
      },
    },
    tasks: {
      async list() {
        return tasks.map(cloneTask);
      },

      async get(id: string) {
        const task = tasks.find((item) => item.id === id);
        if (!task) {
          throw new TaskClientError("NOT_FOUND", "That task no longer exists.");
        }
        return cloneTask(task);
      },

      async create(input: CreateTaskInput) {
        const timestamp = new Date().toISOString();
        const task: Task = {
          id: createTaskId(),
          title: validateTitle(input.title),
          description: input.description?.trim() ?? null,
          status: input.status ?? "todo",
          createdAt: timestamp,
          updatedAt: timestamp,
        };

        tasks = [task, ...tasks];
        return cloneTask(task);
      },

      async update(id: string, input: UpdateTaskInput) {
        const current = tasks.find((task) => task.id === id);
        if (!current) {
          throw new TaskClientError("NOT_FOUND", "That task no longer exists.");
        }

        const updated: Task = {
          ...current,
          ...(input.title === undefined
            ? {}
            : { title: validateTitle(input.title) }),
          ...(input.description === undefined
            ? {}
            : { description: input.description?.trim() ?? null }),
          ...(input.status === undefined ? {} : { status: input.status }),
          updatedAt: new Date().toISOString(),
        };

        tasks = tasks.map((task) => (task.id === id ? updated : task));
        return cloneTask(updated);
      },

      async delete(id: string) {
        if (!tasks.some((task) => task.id === id)) {
          throw new TaskClientError("NOT_FOUND", "That task no longer exists.");
        }

        tasks = tasks.filter((task) => task.id !== id);
      },
    },
  };
}
