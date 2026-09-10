import { useEffect, useState } from "react";
import type { TaskCoreClient } from "@tauri-go-service-example/task-core-library";
import { TaskPage } from "./features/tasks/TaskPage";

interface AppProps {
  client: TaskCoreClient;
  backendLabel: string;
}

const DEFAULT_REFRESH_INTERVAL_MS = 30_000;

function App({ client, backendLabel }: AppProps) {
  const [refreshIntervalMs, setRefreshIntervalMs] = useState(
    DEFAULT_REFRESH_INTERVAL_MS,
  );

  useEffect(() => {
    let active = true;
    client.config
      .getPublic()
      .then((configuration) => {
        if (active) setRefreshIntervalMs(configuration.ui.refreshIntervalMs);
      })
      .catch(() => {
        // Keep the safe built-in fallback; task operations surface their own errors.
      });
    return () => {
      active = false;
    };
  }, [client]);

  return (
    <TaskPage
      client={client}
      backendLabel={backendLabel}
      refreshIntervalMs={refreshIntervalMs}
    />
  );
}

export default App;
