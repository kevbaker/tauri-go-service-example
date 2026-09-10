import { randomBytes } from "node:crypto";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const serviceDirectory = path.join(repositoryRoot, "services/go/task-service");
const configPath = path.join(repositoryRoot, "config/development.yaml");
const token = randomBytes(32).toString("hex");
const remoteClientToken = randomBytes(24).toString("base64url");
const environment = {
  ...process.env,
  TGS_REMOTE_ACCESS_TOKEN: token,
  ...(process.argv.includes("--loopback")
    ? {}
    : { TGS_REMOTE_CLIENT_TOKEN: remoteClientToken }),
};
const npmCommand = process.platform === "win32" ? "npm.cmd" : "npm";
const loopbackOnly = process.argv.includes("--loopback");

const service = spawn("go", ["run", "./cmd/task-service", "--config", configPath], {
  cwd: serviceDirectory,
  env: environment,
  stdio: ["ignore", "pipe", "inherit"],
});

let frontend;
let readiness = "";
let stopping = false;

function stop(exitCode = 0) {
  if (stopping) return;
  stopping = true;
  frontend?.kill("SIGTERM");
  service.kill("SIGTERM");
  process.exitCode = exitCode;
}

service.stdout.setEncoding("utf8");
service.stdout.on("data", (chunk) => {
  process.stdout.write(chunk);
  readiness += chunk;
  const newline = readiness.indexOf("\n");
  if (newline < 0 || frontend) return;

  try {
    const ready = JSON.parse(readiness.slice(0, newline));
    if (ready.protocolVersion !== 1) throw new Error("unsupported protocol version");
  } catch (error) {
    console.error(`Go service readiness failed: ${error.message}`);
    stop(1);
    return;
  }

  if (!loopbackOnly) {
    console.log(
      `Remote UI access: append ?access_token=${encodeURIComponent(remoteClientToken)} to a Vite Network URL.`,
    );
  }

  frontend = spawn(
    npmCommand,
    ["run", loopbackOnly ? "dev" : "dev:remote", "--workspace", "task-app"],
    { cwd: repositoryRoot, env: environment, stdio: "inherit" },
  );
  frontend.on("exit", (code) => stop(code ?? 1));
});

service.on("error", (error) => {
  console.error(`Could not start Go service: ${error.message}`);
  stop(1);
});
service.on("exit", (code) => {
  if (!stopping) {
    console.error(`Go service stopped unexpectedly with exit code ${String(code)}.`);
    stop(code ?? 1);
  }
});

process.on("SIGINT", () => stop(0));
process.on("SIGTERM", () => stop(0));
