---
name: create-specs
description: Creates decomposed specs from free-form scope text. Stops before branching or implementing. Use when the user invokes /create-specs or says "create specs".
allowed-tools: Read, Glob, Grep, Bash, Edit, Write, Agent, Skill
---

# create-specs agent

Gather requirements from the user's prompt, build an implementation plan with user approval at two gates, then save decomposed specs. Never branch or write implementation code.

---

## Phase 0 — Parse input

Examine everything the user passed in the prompt. Identify:

- **`manual_scope`** — the free-form scope description provided. Treat this verbatim.

If the prompt contains no scope description, ask the user:

> Please provide a description of what needs to be built.

Wait for the answer and re-parse.

---

## Phase 1 — Summarize scope → user gate

Synthesize `manual_scope` into a unified **implementation scope summary**:

- **What needs to change** — bullet list of areas/behaviors to modify or add. Business-level, no code.
- **What must stay the same** — existing behavior the change must not break.
- **Open questions** — anything ambiguous or unclear from the provided scope.

For the output folder and plan naming: derive a short slug from `manual_scope` (e.g. `add-refund-cap`) and use that as `<SCOPE-KEY>` throughout.

**Present the scope summary to the user** and ask:

> Does this scope look right? Any corrections or additional context before I plan the implementation?

**Wait for the user's response before proceeding.**

---

## Phase 2 — Implementation plan (interactive) → user gate

### Step 1 — Explore the codebase

Using the scope and the user's Phase 1 response, read the relevant parts of the codebase — controllers, services, models, tests, migrations — wherever the change will land. Do not skim; read enough to know exactly what exists and what needs to change.

### Step 2 — Clarify before committing

Before drafting anything, resolve ambiguities:

- **Ask before assuming.** If uncertain about business logic, ownership, or field meaning — stop and ask. One question at a time; wait for the answer.
- **Surface options when multiple valid approaches exist.** Present each with a 1–2 sentence trade-off and ask which to use.
- **Confirm scope before expanding it.** If implementing correctly requires touching something outside the agreed scope, flag it and ask.

Only proceed to Step 3 once all open questions are answered.

### Step 3 — Draft the plan

Write a numbered list of steps. Each step names the **file(s)** and **what** to change — concrete enough that a different engineer could execute it. Flag explicitly:
- DB migrations needed.
- Frontend changes (React components, NSwag-generated models via `npm run generate-models`).
- Test changes.

After the steps, add:
- **Risks / side-effects**
- **Complexity** — Small / Medium / Large

### Step 4 — Grill the plan

Invoke the **`grilling`** skill via the Skill tool, passing the drafted plan (plus scope summary) as the `args` string. Answer each challenge honestly — defend or concede. Incorporate conceded points directly into the plan; carry unresolved tensions forward as explicit trade-offs or open questions.

### Step 5 — Present for approval

Present the revised plan to the user with these options:

> - **Save** — decompose the plan into specs and save everything under `specs/<SCOPE-KEY>/`, then stop.
> - **Change: \<your edit\>** — revise the plan and re-present.
> - **Cancel** — stop here.

**Do not write any code.** If changes are requested, revise and re-present.

#### Save behaviour

When the user chooses **Save**, spawn an **Agent** with this prompt (fill in the bracketed values from context):

> You are a software engineering manager. Your job is to decompose an implementation plan into a set of independent, self-contained specs — one per logical unit of work — so that any engineer can pick up a single spec and execute it without reading the others.
>
> **Scope key:** `<SCOPE-KEY>` — `<one-line summary>`
> **Scope summary:** `<scope summary from Phase 1>`
> **Implementation plan:**
> ```
> <full numbered plan from Step 3, post-grilling>
> ```
>
> Do the following, in order:
>
> 1. Write `specs/<SCOPE-KEY>/plan.md` — the full implementation plan verbatim, no changes.
> 2. Identify the independent units of work in the plan. A unit is independent when it can be implemented, reviewed, and merged without waiting for another unit from this scope (shared scaffolding that everything else builds on is its own unit). Name each unit clearly.
> 3. For each unit, write `specs/<SCOPE-KEY>/spec-NN-<slug>.md` (zero-padded index, e.g. `spec-01-add-migration.md`) using this structure:
>    - **Goal** — one sentence: what this spec accomplishes and why.
>    - **Context** — what the engineer needs to know before starting (relevant domain rules, existing code to read, assumptions baked into this spec).
>    - **Out of scope** — what is explicitly NOT part of this spec (to prevent over-reach).
>    - **Steps** — numbered, file-level actions identical in detail to the source plan steps, but only the ones belonging to this unit.
>    - **Acceptance criteria** — bullet list of observable outcomes that confirm the spec is done.
>    - **Dependencies** — list any other specs from this scope that must merge first, or "none".
>
> Return a summary listing the files you wrote and one sentence per spec describing what it covers.

After the agent finishes, report the summary to the user. The skill ends here — no branching or implementation.

---

## Guardrails

- **No code before plan approval** — Phase 2 gate is mandatory.
- **No commits, pushes, or branch creation** — leave the working tree untouched.
- **No implementation** — this skill produces specs only.
- **Scope required** — the user must provide a scope description; at least one must be present.
- **Delegate, don't duplicate** — plan critique → `grilling`.
