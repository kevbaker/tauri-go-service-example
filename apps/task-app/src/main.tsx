import React from "react";
import ReactDOM from "react-dom/client";
import "@vaadin/vaadin-lumo-styles/lumo.css";
import App from "./App";
import { createTaskAppRuntime } from "./bridge/app-client";

const runtime = createTaskAppRuntime();

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App client={runtime.client} backendLabel={runtime.backendLabel} />
  </React.StrictMode>,
);
