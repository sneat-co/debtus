# AI agent guidance

This is a **Sneat extension**. Build it against the shared platform standards.

## Building UI (forms, pages, screens, modals, wizards)

Before and while writing UI components, work through the **screen-flow checklist**:
https://github.com/sneat-co/sneat-specs/blob/main/standards/frontend-ux/flows.md
(building-block docs — cards/buttons/lists/forms/modals/states/page-layout — live
alongside it). If the `ui-flow` skill is available, invoke it.

Key rules:
- A screen isn't done until its **entry** (what links here) and **exit** (where it
  sends the user) are wired to real screens. Map the flow before building.
- After a successful **create**, redirect to the new entity's **details** page
  (using the returned id, with `replaceUrl`) — unless explicitly told otherwise.
- Don't leave orphan pages, silent failures, or disconnected wizard steps.

## Extension standards

Backend wiring, frontend apps, and UX:
https://github.com/sneat-co/sneat-libs/blob/main/docs/extension-standards/README.md
