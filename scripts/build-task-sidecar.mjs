import { execFileSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { fileURLToPath } from "node:url";
import path from "node:path";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const serviceDirectory = path.join(repositoryRoot, "services/go/task-service");
const binaryDirectory = path.join(repositoryRoot, "apps/task-app/src-tauri/binaries");
const targetTriple = execFileSync("rustc", ["--print", "host-tuple"], {
  encoding: "utf8",
}).trim();

const target = (() => {
  const architecture = targetTriple.startsWith("aarch64") ? "arm64" : "amd64";
  if (targetTriple.includes("apple-darwin")) return { goos: "darwin", architecture, extension: "" };
  if (targetTriple.includes("windows")) return { goos: "windows", architecture, extension: ".exe" };
  if (targetTriple.includes("linux")) return { goos: "linux", architecture, extension: "" };
  throw new Error(`Unsupported Tauri target: ${targetTriple}`);
})();

mkdirSync(binaryDirectory, { recursive: true });
const output = path.join(
  binaryDirectory,
  `task-service-${targetTriple}${target.extension}`,
);

execFileSync("go", ["build", "-trimpath", "-o", output, "./cmd/task-service"], {
  cwd: serviceDirectory,
  env: {
    ...process.env,
    CGO_ENABLED: "0",
    GOARCH: target.architecture,
    GOOS: target.goos,
  },
  stdio: "inherit",
});

console.log(`Built Go sidecar: ${path.relative(repositoryRoot, output)}`);
