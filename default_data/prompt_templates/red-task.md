# Red task specification
# Filled by the planner for each red task in the plan
# This is the description field the red agent receives

## Title format
[RED] Test: {{TestFunctionName}} — {{one_line_behavior_description}}

## Summary (shown in board card — one sentence max)
Write a failing test verifying that {{behavior_description}}.

## Description (full spec the agent works from)

Write a failing test in `{{test_file_path}}` named `{{TestFunctionName}}`.

### What to test

This test verifies that {{precise_behavior_description}}.

Specifically:
- Input: {{input_description}}
- Expected output: {{expected_output}}
- Expected side effects: {{side_effects_or_none}}

### Test signature

```
{{TestFunctionName}}({{test_parameters_if_any}})
```

### Assertions required

The test must assert:
1. {{assertion_1}}
2. {{assertion_2}}

### Expected failure

When run correctly, the test must fail with:
```
{{expected_failure_output}}
```

It fails because: {{why_it_fails — function does not exist / returns wrong value / interface not satisfied}}

### File location

Test file: `{{test_file_path}}`

If the file already exists, add the test to it. Match the existing style exactly.
If the file does not exist, create it following the naming convention of nearby test files.

### Context

The thing being tested: `{{function_or_type_being_tested}}`
Expected location of implementation: `{{impl_file_path}}`
Relevant existing code to read first: {{context_files}}

### Constraints

- Do not create or modify any implementation file
- Do not write more than this one test
- Match the assertion library and style used in nearby test files
- If a required type or interface does not exist in an implementation file, block — do not create it

## Completion summary format

```
[RED] Test: {{TestFunctionName}} in {{test_file_path}}
Verifies: {{one_sentence_behavior}}
Failure output: {{verbatim_failure_message}}
Failure reason: {{what_does_not_exist_yet}}
To pass: {{one_sentence_of_what_green_must_implement}}
```

## Fields
priority: {{critical|high|medium|low}}
assigned_role: red
feature_id: {{feature_id}}
context_files: [{{file_1}}, {{file_2}}]
tags: [red, {{feature_slug}}]
depends_on: [{{dependency_task_ids}}]
start_in: todo
