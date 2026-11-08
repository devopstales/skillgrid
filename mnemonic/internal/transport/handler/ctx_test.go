package handler

import "context"

// ctx is the fixed test context (the handler is process-scoped in the
// tests; per-call cancellation is a transport concern the queue passes
// through untouched).
var ctx = context.Background()
