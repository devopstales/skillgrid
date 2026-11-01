package tiered

import (
	"context"
	"sync"
)

// ContentWriteHook is the non-blocking seam after an L2 content write.
type ContentWriteHook struct {
	Store *Store
	// WaitGroup is optional; tests may set it to observe background work.
	WaitGroup *sync.WaitGroup
}

// AfterContentWrite schedules GenerateTiers and returns immediately.
// It never awaits summarization. Failures are logged via Store.Logf.
func (h *ContentWriteHook) AfterContentWrite(ctx context.Context, project, fullPath, title string) {
	if h == nil || h.Store == nil {
		return
	}
	wg := h.WaitGroup
	if wg != nil {
		wg.Add(1)
	}
	go func() {
		if wg != nil {
			defer wg.Done()
		}
		if err := h.Store.GenerateTiers(ctx, project, fullPath, title); err != nil {
			h.Store.logf("tiered hook: %v", err)
		}
	}()
}
