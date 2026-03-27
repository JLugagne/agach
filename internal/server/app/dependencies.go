package app

// maxDepth is the maximum allowed depth for task dependency chains.
// Limiting depth prevents O(n) graph traversal attacks via deeply nested dependency chains.
const maxDepth = 20
