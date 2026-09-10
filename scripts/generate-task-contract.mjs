import { mkdir, readFile, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { compile } from "json-schema-to-typescript";

const repositoryRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const schemaPath = resolve(repositoryRoot, "contracts/task-api.schema.json");
const outputPath = resolve(
  repositoryRoot,
  "packages/node/task-core-library/src/generated/task-api.ts",
);
const checkOnly = process.argv.includes("--check");

const schema = JSON.parse(await readFile(schemaPath, "utf8"));
const generated = await compile(schema, "TaskAPIContract", {
  bannerComment:
    "/* This file is generated from contracts/task-api.schema.json. Do not edit. */",
  cwd: repositoryRoot,
  style: {
    singleQuote: false,
  },
});

if (checkOnly) {
  const current = await readFile(outputPath, "utf8").catch(() => "");
  if (current !== generated) {
    console.error(
      "Generated task contract is stale. Run `npm run contract:generate`.",
    );
    process.exitCode = 1;
  }
} else {
  await mkdir(dirname(outputPath), { recursive: true });
  await writeFile(outputPath, generated);
}
