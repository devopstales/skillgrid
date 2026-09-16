package mcp

import (
	"os"
	"sync"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// StorePool is the lazy per-project store pool for the MCP server: stores are
// opened on first query, evicted after inactivity, and bounded in concurrency.
// The optional repo param selects the project; when omitted a single indexed
// project or the MCP default/cwd resolves it.
type StorePool struct {
	mu        sync.Mutex
	entries   map[string]*poolEntry
	dataDir   string
	idle      time.Duration
	maxOpen   int
	now       func() time.Time
}

type poolEntry struct {
	store     *store.Store
	lastUsed  time.Time
	inflight  int32
}

// DefaultIdleEvict is the inactivity window before an idle store is evicted.
const DefaultIdleEvict = 5 * time.Minute

// NewStorePool returns a lazy store pool. idle <= 0 uses DefaultIdleEvict;
// maxOpen <= 0 is unbounded.
func NewStorePool(dataDir string, idle time.Duration, maxOpen int) *StorePool {
	if idle <= 0 {
		idle = DefaultIdleEvict
	}
	return &StorePool{
		entries: map[string]*poolEntry{},
		dataDir: dataDir,
		idle:    idle,
		maxOpen: maxOpen,
		now:     time.Now,
	}
}

// Get opens (lazily) or reuses the store for projectID, recording activity.
// It evicts idle entries first to honor the concurrency bound.
func (p *StorePool) Get(projectID string) (*store.Store, func(), error) {
	if p == nil {
		return nil, nil, errNilPool
	}
	p.evictIdle()
	p.mu.Lock()
	if e, ok := p.entries[projectID]; ok {
		e.lastUsed = p.now()
		p.mu.Unlock()
		st := e.store
		return st, func() { p.release(projectID) }, nil
	}
	p.entries[projectID] = &poolEntry{lastUsed: p.now()}
	p.mu.Unlock()

	st, err := store.Open(p.dataDir, projectID)
	if err != nil {
		p.mu.Lock()
		delete(p.entries, projectID)
		p.mu.Unlock()
		return nil, nil, err
	}
	p.mu.Lock()
	if e, ok := p.entries[projectID]; ok {
		e.store = st
	} else {
		p.entries[projectID] = &poolEntry{store: st, lastUsed: p.now()}
	}
	p.mu.Unlock()
	return st, func() { p.release(projectID) }, nil
}

// release marks a store as free (reference count). It does not close the store;
// eviction closes idle stores.
func (p *StorePool) release(projectID string) {
	p.mu.Lock()
	if e, ok := p.entries[projectID]; ok {
		e.lastUsed = p.now()
	}
	p.mu.Unlock()
}

// evictIdle closes stores idle for longer than the window, honoring the
// maxOpen bound by evicting from the least-recently-used.
func (p *StorePool) evictIdle() {
	now := p.now()
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, e := range p.entries {
		if e.store == nil {
			continue
		}
		if now.Sub(e.lastUsed) > p.idle {
			_ = e.store.Close()
			delete(p.entries, id)
		}
	}
	if p.maxOpen <= 0 {
		return
	}
	open := 0
	var lru []string
	for id, e := range p.entries {
		if e.store == nil {
			continue
		}
		open++
		lru = append(lru, id)
	}
	if open <= p.maxOpen {
		return
	}
	// Sort by lastUsed ascending and evict the oldest until within the bound.
	sortByLRU := func() {
		for i := 0; i < len(lru); i++ {
			for j := i + 1; j < len(lru); j++ {
				if p.entries[lru[j]].lastUsed.Before(p.entries[lru[i]].lastUsed) {
					lru[i], lru[j] = lru[j], lru[i]
				}
			}
		}
	}
	sortByLRU()
	for len(p.entries) > p.maxOpen {
		var victim string
		for id, e := range p.entries {
			if e.store != nil {
				victim = id
				break
			}
		}
		if victim == "" {
			return
		}
		_ = p.entries[victim].store.Close()
		delete(p.entries, victim)
	}
}

// Len returns the number of open stores (test hook).
func (p *StorePool) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := 0
	for _, e := range p.entries {
		if e.store != nil {
			n++
		}
	}
	return n
}

// projectIDForPool resolves the project for a request: an explicit repo param
// wins; otherwise, when a single project is indexed, that one; otherwise the
// cwd-resolved project.
func projectIDForPool(svc *service.Service, repo string) (string, error) {
	if repo != "" {
		return project.NormalizeID(repo), nil
	}
	projects, err := svc.ListProjects()
	if err == nil && len(projects) == 1 {
		return projects[0], nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return svc.ResolveProject(cwd)
}

var errNilPool = errNilPoolValue{}

type errNilPoolValue struct{}

func (errNilPoolValue) Error() string { return "store pool is nil" }
