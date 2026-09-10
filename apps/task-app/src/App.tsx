import type { TaskCoreClient } from "@tauri-go-service-example/task-core-library";
import { TaskPage } from "./features/tasks/TaskPage";

interface AppProps {
  client: TaskCoreClient;
  backendLabel: string;
}

function App({ client, backendLabel }: AppProps) {
  return <TaskPage client={client} backendLabel={backendLabel} />;
}

export default App;
