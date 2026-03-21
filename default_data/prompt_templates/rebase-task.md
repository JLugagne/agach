# Rebase task specification
# Created automatically by the daemon when a feature branch diverges from main
# The daemon fills all fields — this is not written by the planner

## Title format
[REBASE] {{feature.title}} — resolve conflicts with main

## Summary (shown in board card — one sentence max)
Resolve {{conflict_count}} conflict(s) in {{conflict_files_count}} file(s) introduced by changes to main since {{base_commit_short}}.

## Description (full spec the agent works from)

The feature branch for `{{feature.title}}` has diverged from main and cannot be merged cleanly.

Your session has been resumed with the original context of the work that caused the conflicts. You have full memory of what was being implemented and why. Use that memory to resolve intelligently.

### Conflict information

Branch: `{{worktree_branch}}`
Base commit (when feature work started): `{{base_commit}}`
Current main: `{{current_main_commit}}`
Commits added to main since branch: {{commits_since_branch}}

Conflicting files:
{{#each conflict_files}}
- `{{this}}`
{{/each}}

### What changed in main

```
{{git_log_summary}}
```

Key changes relevant to the conflicts:
- {{relevant_change_1}}
- {{relevant_change_2}}

### Original work context

This conflict was introduced by task `{{source_task_id}}` — {{source_task_title}}.

That task's completion summary:
```
{{source_task_completion_summary}}
```

Your session ID `{{session_id}}` was used to produce that work. Your session has been resumed — you remember the implementation decisions.

### Resolution process

1. Review the conflicts: `git diff --diff-filter=U`
2. For each conflict, understand both sides using your session memory and the main changes above
3. Resolve — do not pick sides blindly, understand both intents
4. Stage resolved files: `git add {{conflict_files}}`
5. Continue: `git rebase --continue`
6. Run the full test suite — everything must pass

### If conflicts are unresolvable

If two changes are genuinely incompatible and you cannot determine which takes precedence:

Block: `UNRESOLVABLE CONFLICT — {{file}}: the feature implements <X> because <reason from your memory>. Main changed to <Y> because <reason from diff>. These cannot coexist without a design decision. Which takes precedence?`

## Completion summary format

```
[REBASE] Feature: {{feature.title}}
Conflicts resolved: {{N}} in {{files}}
Resolution approach:
  {{file_1}}: {{how_it_was_resolved}}
  {{file_2}}: {{how_it_was_resolved}}
Main changes integrated:
  {{what_from_main_was_incorporated}}
Full suite: passing
Branch: ready to continue
```

## Fields
priority: critical
assigned_role: rebase
feature_id: {{feature_id}}
context_files: {{conflict_files}}
tags: [rebase, {{feature_slug}}]
depends_on: [{{source_task_id}}]
start_in: todo
session_id: {{session_id}}
