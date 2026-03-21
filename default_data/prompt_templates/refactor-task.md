# Refactor task specification
# Filled by the planner for each refactor task in the plan
# This is the description field the refactor agent receives

## Title format
[REFACTOR] {{what_is_being_refactored}} in {{file_or_package}}

## Summary (shown in board card — one sentence max)
Refactor {{what}} in {{file}} to {{why_in_five_words_or_less}}.

## Description (full spec the agent works from)

Refactor `{{what_specifically}}` in `{{impl_file_path}}`.

### Current state

```
{{current_code_or_description_of_current_structure}}
```

### Target state

```
{{target_code_or_description_of_target_structure}}
```

### Changes in scope

Apply exactly these changes, nothing more:

1. {{change_1_description}}
   - File: `{{file}}`
   - What: {{specific_change}}

2. {{change_2_description}}
   - File: `{{file}}`
   - What: {{specific_change}}

### Files in scope

Only these files may be modified:
- `{{file_1}}`
- `{{file_2}}`

No other files are in scope. If you notice something in another file that also needs improvement, do not fix it — note it in the completion summary.

### What must not change

- All existing tests must pass before and after
- The public API / function signatures must not change
- {{other_invariant_1}}
- {{other_invariant_2}}

### Reason for this refactor

{{why_this_improves_the_codebase — not "clean code" but specific: "extracted from CompleteTask because it is now needed by three callers" or "renamed because the original name was misleading after the behavior changed in G3"}}

### Context

Paired green task: {{green_task_id}} — read its completion summary to understand what was just implemented.
Relevant existing code: {{context_files}}

### Verification

Run the full test suite before making any changes. If anything is failing, block — do not proceed.

After each meaningful change, run:
```
{{test_command}}
```

Run the full suite once more after all changes are complete. Everything must pass.

## Completion summary format

```
[REFACTOR] Refactored: {{what_was_changed}}
Files: {{list_of_files_modified}}
Changes:
  - {{change_1_description}}
  - {{change_2_description}}
Reason: {{why_this_improves_the_codebase}}
Noted for future: {{anything_noticed_but_out_of_scope}}
Full suite: passing
```

## Fields
priority: {{critical|high|medium|low}}
assigned_role: refactor
feature_id: {{feature_id}}
context_files: [{{file_1}}, {{file_2}}]
tags: [refactor, {{feature_slug}}]
depends_on: [{{green_task_id}}]
start_in: todo
