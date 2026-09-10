import { useEffect, useRef, useState, type FormEvent } from "react";
import {
  Button,
  Select,
  TextArea,
  TextField,
} from "@vaadin/react-components";
import type {
  CreateTaskInput,
  Task,
  TaskStatus,
} from "@tauri-go-service-example/task-core-library";

const statusItems = [
  { label: "To do", value: "todo" },
  { label: "In progress", value: "in_progress" },
  { label: "Done", value: "done" },
];

interface TaskFormProps {
  editingTask: Task | null;
  busy: boolean;
  onCancel(): void;
  onSubmit(input: CreateTaskInput): Promise<void>;
}

export function TaskForm({
  editingTask,
  busy,
  onCancel,
  onSubmit,
}: TaskFormProps) {
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<TaskStatus>("todo");
  const [titleError, setTitleError] = useState("");
  const formRef = useRef<HTMLFormElement>(null);

  useEffect(() => {
    setTitle(editingTask?.title ?? "");
    setDescription(editingTask?.description ?? "");
    setStatus(editingTask?.status ?? "todo");
    setTitleError("");
  }, [editingTask]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!title.trim()) {
      setTitleError("Enter a task title");
      return;
    }

    setTitleError("");
    await onSubmit({ title, description, status });

    if (!editingTask) {
      setTitle("");
      setDescription("");
      setStatus("todo");
    }
  }

  return (
    <form ref={formRef} className="task-form" onSubmit={handleSubmit}>
      <div className="task-form__heading">
        <div>
          <p className="eyebrow">{editingTask ? "Edit task" : "New task"}</p>
          <h2>{editingTask ? editingTask.title : "What needs doing?"}</h2>
        </div>
        {editingTask ? (
          <Button theme="tertiary" onClick={onCancel} disabled={busy}>
            Cancel
          </Button>
        ) : null}
      </div>

      <div className="task-form__fields">
        <TextField
          label="Title"
          value={title}
          maxlength={120}
          required
          invalid={Boolean(titleError)}
          errorMessage={titleError}
          onValueChanged={(event) => setTitle(event.detail.value)}
        />
        <Select
          label="Status"
          value={status}
          items={statusItems}
          onValueChanged={(event) => setStatus(event.detail.value as TaskStatus)}
        />
        <TextArea
          className="task-form__description"
          label="Description"
          value={description}
          maxlength={500}
          onValueChanged={(event) => setDescription(event.detail.value)}
        />
      </div>

      <div className="task-form__actions">
        <Button
          theme="primary"
          disabled={busy}
          onClick={() => formRef.current?.requestSubmit()}
        >
          {busy ? "Saving…" : editingTask ? "Save changes" : "Add task"}
        </Button>
      </div>
    </form>
  );
}
