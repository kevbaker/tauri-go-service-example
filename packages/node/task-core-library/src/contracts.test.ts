import { describe, expect, it } from "vitest";

import taskSchema from "../../../../contracts/task-api.schema.json";
import { taskOperations, taskStatuses } from "./contracts";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readDefinitionEnum(name: string): string[] {
  const schema: unknown = taskSchema;
  if (!isRecord(schema) || !isRecord(schema.$defs)) {
    throw new Error("Task schema is missing $defs.");
  }
  const definition = schema.$defs[name];
  if (
    !isRecord(definition) ||
    !Array.isArray(definition.enum) ||
    !definition.enum.every((value) => typeof value === "string")
  ) {
    throw new Error(`Task schema definition ${name} is not a string enum.`);
  }
  return definition.enum;
}

describe("generated task contract", () => {
  it("keeps runtime status and operation values aligned with the schema", () => {
    expect(taskStatuses).toEqual(readDefinitionEnum("TaskStatus"));
    expect(taskOperations).toEqual(readDefinitionEnum("TaskOperation"));
  });
});
