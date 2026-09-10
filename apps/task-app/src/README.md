# Frontend Source

React and TypeScript source belongs here. Follow the feature-oriented layout in the repository's [TypeScript and React patterns](../../../.agents/patterns/typescript-react.md).

The responsive Vaadin task CRUD screen is implemented. Task components consume `TaskCoreClient` and task types from [`packages/node/task-core-library`](../../../packages/node/task-core-library/README.md); transport construction remains in `src/bridge` and the application composition root.
