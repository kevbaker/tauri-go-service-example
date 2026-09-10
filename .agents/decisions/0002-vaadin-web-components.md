# ADR 0002: Vaadin Web Components for the task UI

- Status: accepted
- Date: 2026-09-09

## Context

The POC needs an open-source design system for its React and TypeScript UI. The task CRUD surface must work in both Tauri's desktop webview and a normal browser, adapt to an iPhone-sized viewport, and avoid creating a bespoke component library.

Vaadin publishes framework-independent Web Components and official React wrappers. Its form controls provide labels, validation states, keyboard behavior, and Lumo design tokens. Vaadin documents responsive layouts as a combination of adaptive components and CSS media or container queries.

## Decision

Use Vaadin Web Components through the official `@vaadin/react-components` wrappers, with `@vaadin/vaadin-lumo-styles` providing the Lumo theme.

- Use Vaadin controls for interactive form elements and actions.
- Use semantic HTML and small application-owned CSS for page structure and responsive task cards.
- Design mobile-first at a 320-pixel minimum width and verify the current iPhone viewport explicitly.
- Keep controls at least 44 pixels high for primary touch interactions.
- Use one column on phones, a wider form layout on tablets, and a split form/list workspace only on large screens.
- Do not introduce Vaadin Flow, Hilla, or a Java server. The application remains a Vite/React frontend talking to the typed `TaskCoreClient`.

The initial frontend uses an in-memory `TaskCoreClient` solely to make the CRUD interface executable before the desktop and HTTP clients are connected. It does not claim persistence; the UI labels this state and resets on reload. The in-memory implementation will later be replaced at the application composition root by the shared core client plus `DesktopTransport` or `HttpTransport` without changing task components.

## Consequences

- Mantine is removed from dependencies and documentation.
- The UI uses Vaadin's shipped component behavior while remaining deployable as ordinary static frontend assets.
- Vaadin component styles add bundle weight, which is acceptable for this communication-model POC and should be measured rather than optimized speculatively.
- Application CSS still owns responsive page composition because a compact card list is clearer than a desktop data grid on an iPhone.
- Browser and Tauri rendering must both be tested because they may use different webview engines.

## Sources

- [Vaadin components](https://vaadin.com/docs/latest/components)
- [Vaadin Form Layout](https://vaadin.com/docs/latest/components/form-layout)
- [Vaadin responsive layouts](https://vaadin.com/docs/latest/building-apps/ui-basics/layouts/responsive)
