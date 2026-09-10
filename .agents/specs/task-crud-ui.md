# Task CRUD UI Specification

- Status: frontend prototype implemented; Go-backed UI integration pending
- Date: 2026-09-09
- Related decision: [ADR 0002](../decisions/0002-vaadin-web-components.md)

## Purpose

Provide the smallest useful UI that exercises create, read, update, status-change, validation, and delete interactions through the shared `TaskCoreClient`. The task domain is a test fixture for the bridge, not a product feature.

## Current behavior

- Show total, open, and completed task counts.
- List tasks as compact, single-line cards with a title, description tooltip, status, and actions.
- Create a title-only task; new tasks start as Pending.
- Edit the title, optional description, and status using the shared form.
- Change status from the edit form or cycle Pending, In progress, and Done directly from a task card's status pill.
- Confirm before deletion.
- Display validation and normalized application errors without exposing transport details.
- Clearly label the current adapter as an in-memory preview that resets on reload.

All task operations go through `TaskCoreClient` from `packages/node/task-core-library`. The initial in-memory implementation is selected only in `main.tsx`; task features do not import Tauri APIs or call HTTP.

## Responsive behavior

- Support viewports down to 320 CSS pixels without horizontal scrolling.
- Use a single-column flow on phones so form fields and task cards retain readable width.
- Make primary and card action controls at least 44 pixels high.
- Respect iPhone safe-area insets and dynamic viewport height.
- Allow long titles and descriptions to wrap instead of widening the page.
- Expand the form to two columns at 640 pixels and use a form/list split only at 920 pixels.
- Preserve semantic headings, labels, focus indicators, error announcements, and reduced-motion preferences.

## Verified frontend state

On 2026-09-09 the browser implementation was checked at a 393 by 852 CSS-pixel viewport:

- Vaadin controls rendered with the Lumo theme.
- The page width and scroll width were both 393 pixels.
- A second 320-pixel-wide check also had no horizontal overflow.
- Creating a task increased the summary and displayed the new card.
- Editing the created task changed its visible title.
- Changing its status to done updated the card and summary counts.

Deletion is implemented with a confirmation prompt but was not exercised during this browser check. Persistence, transport parity, and Tauri-webview rendering remain unverified until their respective implementations exist.

## Acceptance criteria for the completed POC

- Repeat the CRUD checks through both `DesktopTransport` and `HttpTransport`.
- Reload and confirm Go/SQLite persistence rather than an in-memory reset.
- Verify the narrowest supported iPhone viewport and at least one desktop viewport.
- Verify keyboard-only creation, editing, status changes, cancellation, and deletion.
- Verify empty, loading, validation-error, service-unavailable, and unexpected-error states.
- Verify the packaged Tauri application in addition to browser development mode.
