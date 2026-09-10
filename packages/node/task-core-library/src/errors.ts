export type TaskClientErrorCode =
  | "VALIDATION"
  | "NOT_FOUND"
  | "CONFLICT"
  | "INTERNAL"
  | "SERVICE_UNAVAILABLE"
  | "PROTOCOL_ERROR";

export interface TaskClientErrorOptions {
  requestId?: string;
  fieldErrors?: Readonly<Record<string, string>>;
}

export class TaskClientError extends Error {
  readonly code: TaskClientErrorCode;
  readonly requestId?: string;
  readonly fieldErrors?: Readonly<Record<string, string>>;

  constructor(
    code: TaskClientErrorCode,
    message: string,
    options: TaskClientErrorOptions = {},
  ) {
    super(message);
    this.name = "TaskClientError";
    this.code = code;
    if (options.requestId !== undefined) this.requestId = options.requestId;
    if (options.fieldErrors !== undefined) this.fieldErrors = options.fieldErrors;
  }
}
