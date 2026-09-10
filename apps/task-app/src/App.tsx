import type { TaskCoreClient } from "@tauri-go-service-example/task-core-library";
import { TaskPage } from "./features/tasks/TaskPage";

interface AppProps {
  client: TaskCoreClient;
}

function App({ client }: AppProps) {
  return <TaskPage client={client} />;
}

export default App;
