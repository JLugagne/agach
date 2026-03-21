# Feature specification template
# Used by Claude Code when calling create_feature via MCP
# Claude Code fills this out through conversation with the user
# then calls create_feature with the structured output

title: >
  Short, verb-first label for the feature.
  Examples: "Add rate limiting", "Implement JWT refresh", "Migrate user table to UUIDv7"

description: >
  The original user request verbatim. Do not paraphrase.
  This is the source of truth for what was asked.

acceptance_criteria:
  # Each criterion must be testable — an agent must be able to write a test for it.
  # Bad:  "The system handles errors gracefully"
  # Good: "When an invalid token is provided, the API returns 401 with body {error: 'invalid_token'}"
  - >
    {{criterion_1}}
  - >
    {{criterion_2}}
  - >
    {{criterion_3}}

out_of_scope:
  # Explicit list of things this feature does NOT cover.
  # Forces the conversation to clarify boundaries before planning begins.
  # Bad:  "future improvements"
  # Good: "per-endpoint rate limits (global only for now)"
  - >
    {{out_of_scope_1}}
  - >
    {{out_of_scope_2}}

constraints:
  # Technical or business constraints the implementation must respect.
  # Examples: "must not change the public API signature", "must be backward compatible",
  #           "must not add new database tables", "response time must stay under 100ms"
  - >
    {{constraint_1}}

open_questions:
  # Anything that could not be resolved in the conversation.
  # These become blocked tasks automatically.
  # Each question must be specific enough that a one-sentence answer resolves it.
  # Bad:  "How should errors be handled?"
  # Good: "Should rate limit errors return 429 or 503? RFC 7231 recommends 429 with Retry-After."
  - >
    {{question_1}}

# Fields set automatically by the system — do not fill in
# feature_id: auto-generated UUIDv7
# project_id: from context
# status: defining
# created_at: now
# spent_usd: 0

# Optional — set if the user specified a budget
# budget_usd: null
