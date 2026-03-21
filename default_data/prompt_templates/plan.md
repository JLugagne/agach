# Plan: {{feature.title}}

**Feature ID:** {{feature.id}}
**Project:** {{project.name}}
**Generated:** {{date}}

---

## Acceptance criteria

{{#each feature.acceptance_criteria}}
- [ ] {{this}}
{{/each}}

---

## Assumptions

> List any assumptions made during decomposition that are not explicit in the feature spec.
> If any of these are wrong, the plan needs revision.

- {{assumption_1}}
- {{assumption_2}}

---

## Open questions

> List anything that remains unclear. These will become blocked tasks.
> Human must answer before the plan is committed.

- [ ] {{question_1}}
- [ ] {{question_2}}

---

## Task tree

> Read this top to bottom. Each pair is a red (R) → green (G) sequence.
> Dependencies are shown with ↳. A task may not start until all tasks it depends on are done.
> Indentation shows dependency depth.

---

### Critical path

```
[R1] {{red_task_1_title}}                                    critical  red
  ↳ [G1] {{green_task_1_title}}                             critical  green
      ↳ [RF1] {{refactor_task_1_title}}                     medium    refactor
          ↳ [R2] {{red_task_2_title}}                       critical  red
              ↳ [G2] {{green_task_2_title}}                 critical  green
```

---

### Full task list

| # | Title | Type | Priority | Role | Depends on |
|---|-------|------|----------|------|------------|
| R1 | {{red_task_1_title}} | red | critical | red | — |
| G1 | {{green_task_1_title}} | green | critical | green | R1 |
| RF1 | {{refactor_task_1_title}} | refactor | medium | refactor | G1 |
| R2 | {{red_task_2_title}} | red | critical | red | G1 |
| G2 | {{green_task_2_title}} | green | critical | green | R2 |
| R3 | {{red_task_3_title}} | red | high | red | G1 |
| G3 | {{green_task_3_title}} | green | high | green | R3 |
| RV | Review: {{feature.title}} | reviewer | low | reviewer | G1, G2, G3 |

---

### Task descriptions

> Full description for each task. This is what the agent will receive.
> Review each one carefully — the agent works from this description alone.

---

#### R1 — {{red_task_1_title}}

**Type:** red
**Priority:** critical
**Assigned role:** red
**Depends on:** —
**Context files:** `{{file_1}}`, `{{file_2}}`
**Tags:** {{tags}}

**Description:**

Write a failing test in `{{test_file_path}}` named `{{test_function_name}}`.

This test verifies that {{behavior_description}}.

The test must:
- Call `{{function_or_method_signature}}`
- Assert {{expected_outcome}}
- Fail with: `{{expected_failure_message}}`

The test must fail because {{why_it_fails_now}}.

---

#### G1 — {{green_task_1_title}}

**Type:** green
**Priority:** critical
**Assigned role:** green
**Depends on:** R1
**Context files:** `{{impl_file_path}}`

**Description:**

Make `{{test_function_name}}` in `{{test_file_path}}` pass.

Implement `{{function_or_method_signature}}` in `{{impl_file_path}}`.

The implementation must:
- {{constraint_1}}
- {{constraint_2}}

Do not modify the test. Do not add behavior the test does not require.

---

#### RF1 — {{refactor_task_1_title}}

**Type:** refactor
**Priority:** medium
**Assigned role:** refactor
**Depends on:** G1
**Context files:** `{{impl_file_path}}`

**Description:**

Refactor `{{what}}` in `{{impl_file_path}}`.

Current state: {{current_state}}
Target state: {{target_state}}

Changes in scope:
- {{change_1}}
- {{change_2}}

Files in scope: `{{file_1}}`, `{{file_2}}`

Do not change behavior. All tests must pass before and after.

---

#### RV — Review: {{feature.title}}

**Type:** reviewer
**Priority:** low
**Assigned role:** reviewer
**Depends on:** G1, G2, G3 (all green and refactor tasks)

**Description:**

Verify that the completed implementation of {{feature.title}} satisfies all acceptance criteria.

Acceptance criteria to validate:
{{#each feature.acceptance_criteria}}
- [ ] {{this}}
{{/each}}

Create follow-up tasks for anything missing. Do not fix anything directly.

---

## Summary

| Metric | Count |
|--------|-------|
| Red tasks | {{red_count}} |
| Green tasks | {{green_count}} |
| Refactor tasks | {{refactor_count}} |
| Reviewer tasks | 1 |
| **Total** | **{{total_count}}** |

**Estimated token budget:** {{estimated_budget}}
**Critical path length:** {{critical_path_length}} tasks

---

## Approval

> Review the task descriptions above carefully.
> When ready, reply: "approved" to commit this plan to the board.
> To request changes: describe what needs to change and the planner will revise.

- [ ] Task descriptions are precise enough for agents to work from
- [ ] Dependency order is correct
- [ ] No work is planned that is already done
- [ ] Open questions above have been answered or are acceptable as blocked tasks
