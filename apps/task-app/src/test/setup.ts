import "@testing-library/jest-dom/vitest";

import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

// Vaadin registers typed custom properties when its components are imported.
// jsdom intentionally does not implement this browser rendering API.
Object.defineProperty(globalThis.CSS, "registerProperty", {
  configurable: true,
  value: () => undefined,
});
Object.defineProperty(globalThis.CSS, "supports", {
  configurable: true,
  value: () => false,
});

class TestResizeObserver implements ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
}

globalThis.ResizeObserver = TestResizeObserver;

afterEach(cleanup);
