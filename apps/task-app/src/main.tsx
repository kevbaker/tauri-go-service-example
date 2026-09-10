import React from "react";
import ReactDOM from "react-dom/client";
import "@vaadin/vaadin-lumo-styles/lumo.css";
import App from "./App";
import { createInMemoryAppClient } from "./bridge/in-memory-app-client";

const client = createInMemoryAppClient();

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App client={client} />
  </React.StrictMode>,
);
