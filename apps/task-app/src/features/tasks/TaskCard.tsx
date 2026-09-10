import "@vaadin/icons/vaadin-iconset.js";
import { Button, Icon } from "@vaadin/react-components";
import type {
  Task,
  TaskStatus,
} from "@tauri-go-service-example/task-core-library";

const statusItems = [
  { label: "Pending", value: "todo" },
  { label: "In progress", value: "in_progress" },
  { label: "Done", value: "done" },
] as const;

function nextStatus(status: TaskStatus): TaskStatus {
  const currentIndex = statusItems.findIndex((item) => item.value === status);
  return statusItems[(currentIndex + 1) % statusItems.length]?.value ?? "todo";
}

interface TaskCardProps {
  task: Task;
  busy: boolean;
  onDelete(task: Task): Promise<void>;
  onEdit(task: Task): void;
  onStatusChange(task: Task, status: TaskStatus): Promise<void>;
}

export function TaskCard({
  task,
  busy,
  onDelete,
  onEdit,
  onStatusChange,
}: TaskCardProps) {
  const status = statusItems.find((item) => item.value === task.status);
  const next = statusItems.find((item) => item.value === nextStatus(task.status));

  return (
    <article className="task-card">
      <div className="task-card__body" title={task.description || undefined}>
        <h3>{task.title}</h3>
      </div>

      <div className="task-card__actions">
        <Button
          className="status-pill"
          data-status={task.status}
          theme="tertiary small"
          disabled={busy}
          aria-label={`Status: ${status?.label}. Change to ${next?.label}`}
          onClick={() => void onStatusChange(task, nextStatus(task.status))}
        >
          {status?.label}
        </Button>
        <Button
          className="task-card__icon-button"
          theme="tertiary"
          disabled={busy}
          aria-label={`Edit ${task.title}`}
          onClick={() => onEdit(task)}
        >
          <Icon icon="vaadin:edit" />
        </Button>
        <Button
          className="task-card__icon-button"
          theme="tertiary error"
          disabled={busy}
          aria-label={`Delete ${task.title}`}
          onClick={() => void onDelete(task)}
        >
          <Icon icon="vaadin:trash" />
        </Button>
      </div>
    </article>
  );
}
