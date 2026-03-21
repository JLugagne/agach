# Green task specification
# Filled by the planner for each green task in the plan
# This is the description field the green agent receives

## Title format
[GREEN] Implement: {{FunctionOrMethodName}} — {{one_line_behavior_description}}

## Summary (shown in board card — one sentence max)
Make {{TestFunctionName}} pass by implementing {{function_or_method}}.

## Description (full spec the agent works from)

Make the test `{{TestFunctionName}}` in `{{test_file_path}}` pass.

### What to implement

Implement `{{function_or_method_signature}}` in `{{impl_file_path}}`.

### Contract

The implementation must satisfy exactly what the test asserts:
- When called with {{input_description}}, it must return {{expected_output}}
- {{constraint_1}}
- {{constraint_2}}

### Implementation location

File: `{{impl_file_path}}`

If the file exists, add to it. If it does not exist, create it following the naming and package conventions of nearby files.

### Minimal scope

Implement only what makes `{{TestFunctionName}}` pass. Specifically:

Do implement:
- {{what_to_implement_1}}
- {{what_to_implement_2}}

Do not implement:
- {{what_not_to_implement_1}} — this is covered by a separate task
- {{what_not_to_implement_2}} — out of scope for this task

### Context

Paired red task: {{red_task_id}} — read its completion summary for the exact failure output and what it expects.
Relevant existing code: {{context_files}}
Interfaces to satisfy: {{interface_names_and_locations}}

### Verification

After implementing, run:
```
{{test_command_for_this_test}}
```

Then run the full suite for the affected package to confirm no regressions:
```
{{full_suite_command}}
```

Both must pass before completing.

### Constraints

- Do not modify the test file
- Do not add behavior the test does not require
- Do not refactor — that is a separate task
- If making the test pass requires changes that seem larger than expected, block rather than proceeding

## Completion summary format

```
[GREEN] Implements: {{FunctionOrMethod}} in {{impl_file_path}}
Makes pass: {{TestFunctionName}}
Implementation: {{one_or_two_sentences_describing_what_was_implemented}}
Intentionally not added: {{what_was_deliberately_left_out}}
Files changed: {{list_of_files}}
Full suite: passing
```

## Fields
priority: {{critical|high|medium|low}}
assigned_role: green
feature_id: {{feature_id}}
context_files: [{{file_1}}, {{file_2}}]
tags: [green, {{feature_slug}}]
depends_on: [{{red_task_id}}]
start_in: todo
