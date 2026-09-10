import { Button, Select } from "@vaadin/react-components";
import type {
  Task,
  TaskStatus,
} from "@tauri-go-service-example/task-core-library";

const statusItems = [
  { label: "To do", value: "todo" },
  { label: "In progress", value: "in_progress" },
  { label: "Done", value: "done" },
];

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
  return (
    <article className="task-card">
      <div className="task-card__body">
        <div className="task-card__title-row">
          <h3>{task.title}</h3>
          <span className="status-pill" data-status={task.status}>
            {statusItems.find((item) => item.value === task.status)?.label}
          </span>
        </div>
        {task.description ? <p>{task.description}</p> : null}
      </div>

      <div className="task-card__controls">
        <Select
          label="Status"
          value={task.status}
          items={statusItems}
          disabled={busy}
          onValueChanged={(event) =>
            void onStatusChange(task, event.detail.value as TaskStatus)
          }
        />
        <div className="task-card__actions">
          <Button theme="tertiary" disabled={busy} onClick={() => onEdit(task)}>
            Edit
          </Button>
          <Button
            theme="tertiary error"
            disabled={busy}
            onClick={() => void onDelete(task)}
          >
            Delete
          </Button>
        </div>
      </div>
    </article>
  );
}
