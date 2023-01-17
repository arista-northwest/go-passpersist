package passpersist

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/rs/zerolog/log"
)

func NewCache() *Cache {
	return &Cache{
		staged:    make(map[string]*VarBind),
		committed: make(map[string]*VarBind),
	}
}

type Cache struct {
	staged    map[string]*VarBind
	committed map[string]*VarBind
	index     []OID
	mu        sync.RWMutex
}

func (c *Cache) Commit() error {
	c.Lock()
	defer c.Unlock()

	c.committed = c.staged
	c.staged = make(map[string]*VarBind)

	c.reIndex()

	return nil
}

func (c *Cache) reIndex() {

	idx := make([]OID, len(c.committed))
	var i int
	for _, v := range c.committed {
		idx[i] = v.Oid
		i++
	}

	sort.Slice(idx, func(i int, j int) bool {
		for k := range idx[i] {
			if idx[i][k] < idx[j][k] {
				return true
			}
		}
		return false
	})

	c.index = idx
}

func (c *Cache) Dump() {
	c.RLock()
	defer c.RUnlock()

	out := make(map[string]interface{})
	out["staged"] = c.staged
	out["commited"] = c.committed
	out["index"] = c.index

	y, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(y))
}

func (c *Cache) Get(oid OID) *VarBind {
	c.RLock()
	defer c.RUnlock()

	log.Debug().Msgf("getting value at: %s", oid.String())
	if v, ok := c.committed[oid.String()]; ok {
		return v
	}
	return nil
}

func (c *Cache) GetNext(oid OID) *VarBind {
	c.RLock()
	defer c.RUnlock()

	log.Debug().Msgf("getting next value after: %s", oid)

	first := c.index[0]
	if len(oid) < len(first) {
		return c.committed[first.String()]
	}

	var next int
	for i, o := range c.index {
		next = i + 1
		if o.Equal(oid) && next < len(c.index) {
			n := c.index[next]
			return c.committed[n.String()]
		}
	}
	return nil
}

func (c *Cache) Set(v *VarBind) error {

	log.Debug().Msgf("staging: %s %s %v", v.Oid, v.Value.TypeString(), v.Value)

	c.staged[v.Oid.String()] = v

	return nil
}

func (c *Cache) Lock() {
	c.mu.Lock()
}

func (c *Cache) RLock() {
	c.mu.RLock()
}

func (c *Cache) Unlock() {
	c.mu.Unlock()
}

func (c *Cache) RUnlock() {
	c.mu.RUnlock()
}
