import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
// @ts-expect-error type error without @types/node package
import { timingSafeEqual } from "node:crypto";
// @ts-expect-error type error without @types/node package
import process from "node:process";
const host = process.env.TAURI_DEV_HOST;

const REMOTE_ACCESS_COOKIE = "task_remote_access";

function secretsMatch(left: string, right: string): boolean {
  const encoder = new TextEncoder();
  const leftBytes = encoder.encode(left);
  const rightBytes = encoder.encode(right);
  return (
    leftBytes.byteLength === rightBytes.byteLength &&
    timingSafeEqual(leftBytes, rightBytes)
  );
}

function decodeCookie(value: string): string {
  try {
    return decodeURIComponent(value);
  } catch {
    return "";
  }
}

function remoteAccessPlugin(accessToken: string | undefined) {
  return {
    name: "task-remote-access",
    configureServer(server: { middlewares: { use: Function } }) {
      if (!accessToken) return;

      server.middlewares.use(
        (
          request: { headers: { cookie?: string }; method?: string; url?: string },
          response: {
            end(body?: string): void;
            setHeader(name: string, value: string): void;
            statusCode: number;
          },
          next: () => void,
        ) => {
          const requestUrl = new URL(request.url ?? "/", "http://localhost");
          const suppliedToken = requestUrl.searchParams.get("access_token");
          if (
            request.method === "GET" &&
            suppliedToken !== null &&
            secretsMatch(suppliedToken, accessToken)
          ) {
            requestUrl.searchParams.delete("access_token");
            response.statusCode = 303;
            response.setHeader(
              "Set-Cookie",
              `${REMOTE_ACCESS_COOKIE}=${encodeURIComponent(accessToken)}; HttpOnly; SameSite=Strict; Path=/`,
            );
            response.setHeader(
              "Location",
              `${requestUrl.pathname}${requestUrl.search}${requestUrl.hash}`,
            );
            response.setHeader("Cache-Control", "no-store");
            response.end();
            return;
          }

          const authenticated = (request.headers.cookie ?? "")
            .split(";")
            .map((cookie) => cookie.trim().split("=", 2))
            .some(
              ([name, value]) =>
                name === REMOTE_ACCESS_COOKIE &&
                value !== undefined &&
                secretsMatch(decodeCookie(value), accessToken),
            );
          if (authenticated) {
            next();
            return;
          }

          response.statusCode = 401;
          response.setHeader("Content-Type", "text/plain; charset=utf-8");
          response.setHeader("Cache-Control", "no-store");
          response.end(
            "Remote task app authentication required. Open the access URL printed by npm run dev:remote.\n",
          );
        },
      );
    },
  };
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const environment = { ...loadEnv(mode, process.cwd(), ""), ...process.env };
  const serviceToken = environment.TGS_REMOTE_ACCESS_TOKEN;
  const remoteClientToken = environment.TGS_REMOTE_CLIENT_TOKEN;
  const serviceUrl = environment.TGS_TASK_SERVICE_URL ?? "http://127.0.0.1:8787";

  return {
    plugins: [remoteAccessPlugin(remoteClientToken), react()],

  // Vite options tailored for Tauri development and only applied in `tauri dev` or `tauri build`
  //
  // 1. prevent Vite from obscuring rust errors
    clearScreen: false,
    // 2. tauri expects a fixed port, fail if that port is not available
    server: {
      port: 1420,
      strictPort: true,
      host: host || false,
      hmr: host
        ? {
            protocol: "ws",
            host,
            port: 1421,
          }
        : undefined,
      watch: {
        // 3. tell Vite to ignore watching `src-tauri`
        ignored: ["**/src-tauri/**"],
      },
      proxy: serviceToken
        ? {
            "/task-service": {
              target: serviceUrl,
              rewrite: (path) => path.replace(/^\/task-service/, ""),
              configure(proxy) {
                proxy.on("proxyReq", (proxyRequest) => {
                  proxyRequest.removeHeader("origin");
                  proxyRequest.setHeader(
                    "Authorization",
                    `Bearer ${serviceToken}`,
                  );
                });
              },
            },
          }
        : undefined,
    },
  };
});
