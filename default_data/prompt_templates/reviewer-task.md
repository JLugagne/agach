# Reviewer task specification
# One reviewer task per feature, depending on all green and refactor tasks
# This is the description field the reviewer agent receives

## Title format
[REVIEW] {{feature.title}}

## Summary (shown in board card — one sentence max)
Verify that the implementation of {{feature.title}} satisfies all acceptance criteria.

## Description (full spec the agent works from)

Verify the completed implementation of feature `{{feature_id}}` — {{feature.title}}.

### What to verify

Call `get_feature` with feature ID `{{feature_id}}` to read the full specification.

The acceptance criteria to validate:
{{#each feature.acceptance_criteria}}
- [ ] {{this}}
{{/each}}

Out of scope — do not flag these as missing:
{{#each feature.out_of_scope}}
- {{this}}
{{/each}}

### How to verify

1. Run the full test suite. Note every failure.
2. For each acceptance criterion:
   - Find the test that covers it
   - Confirm the test actually verifies the behavior (not just that it passes)
   - Confirm the implementation satisfies the criterion in practice
3. Check integration — does the new code work correctly with what it touches?
4. Check error handling — does it match what was specified in constraints?

### Completed work to review

These tasks were completed for this feature. Read their summaries:
{{#each dependency_task_ids}}
- {{this}}
{{/each}}

### Decision rules

**Approve if:**
- All tests pass
- All acceptance criteria are satisfied
- No obvious integration issues

**Create follow-up tasks if:**
- A test fails → `green` task (or `red`+`green` if behavior is missing)
- A criterion is not covered → `red`+`green` pair
- A quality issue exists → `refactor` task

**Block if:**
- A systemic problem exists that cannot be fixed with targeted tasks
- The approach is fundamentally wrong

### Follow-up task requirements

Every follow-up task you create must be as precise as a planner task:
- Exact test name and file for red tasks
- Exact function and file for green tasks
- Exact refactor scope for refactor tasks

Vague follow-up tasks are not acceptable.

## Completion summary format

```
[REVIEW] Feature: {{feature.title}}
Tests: {{N}} passing, {{N}} failing
  Failing: {{list_or_none}}

Acceptance criteria:
  {{criterion_1}}: SATISFIED / MISSING / PARTIALLY
  {{criterion_2}}: SATISFIED / MISSING / PARTIALLY

Findings:
  {{finding_1_or_none}}
  {{finding_2_or_none}}

Follow-up tasks created:
  {{task_id}}: {{title}} ({{type}}, {{priority}})
  or: none

Verdict: APPROVED | APPROVED WITH FOLLOW-UPS | BLOCKED — SYSTEMIC ISSUE
```

## Fields
priority: low
assigned_role: reviewer
feature_id: {{feature_id}}
context_files: []
tags: [reviewer, {{feature_slug}}]
depends_on: [{{all_green_and_refactor_task_ids}}]
start_in: todo
