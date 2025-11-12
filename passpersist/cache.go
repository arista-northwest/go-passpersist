package passpersist

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
)

func NewCache() *Cache {
	return &Cache{
		staged:    make(map[string]*VarBind),
		committed: make(map[string]*VarBind),
	}
}

type Cache struct {
	sync.RWMutex
	staged    map[string]*VarBind
	committed map[string]*VarBind
	index     OIDs
}

func (c *Cache) getIndex(o OID) (int, bool) {
	for p, v := range c.index {
		if v.Equal(o) {
			return p, true
		}
	}

	for p, v := range c.index {
		if v.StartsWith(o) {
			return p - 1, true
		}
	}
	return 0, false
}

func (c *Cache) Commit() {
	c.Lock()
	defer c.Unlock()

	slog.Debug("commiting cache...")
	c.committed = c.staged
	c.staged = make(map[string]*VarBind)

	// rebuild index
	idx := make(OIDs, 0, len(c.committed))
	for _, vb := range c.committed {
		idx = append(idx, vb.OID)
	}
	c.index = idx.Sort()

	//return nil
}

func (c *Cache) DumpIndex() {
	c.RLock()
	defer c.RUnlock()

	slog.Debug("dumping cache index...")
	slog.Debug("index:", slog.Any("index", c.index))
	y, _ := json.MarshalIndent(c.index, "", "  ")
	fmt.Println(string(y))
}

func (c *Cache) Dump() {
	c.RLock()
	defer c.RUnlock()

	o, _ := json.MarshalIndent(c.committed, "", "  ")
	fmt.Println(string(o))
}

func (c *Cache) Get(oid OID) *VarBind {
	c.RLock()
	defer c.RUnlock()

	slog.Debug("getting value at oid", "oid", oid.String())
	if v, ok := c.committed[oid.String()]; ok {
		slog.Debug("got value", "oid", oid.String(), "value", v.Value.String())
		return v
	}
	return nil
}

func (c *Cache) GetNext(oid OID) *VarBind {
	c.RLock()
	defer c.RUnlock()

	slog.Debug("getting next value after", "oid", oid.String())

	idx, found := c.getIndex(oid)
	slog.Debug("got index at", "oid", oid.String(), "index", idx)
	if !found {
		slog.Debug("index not found", "oid", oid.String())
		return nil
	}

	nidx := idx + 1

	slog.Debug("getting next after", "idx", idx, "next-idx", nidx)

	if nidx < len(c.index) {
		next := c.index[nidx]
		if v, ok := c.committed[next.String()]; ok {
			slog.Debug("got next entry", "oid", next.String(), "val", v.Marshal())
			return v
		} else {
			slog.Debug("no entry for oid", "oid", next.String())
		}
	} else {
		slog.Debug("index out of bounds", "idxLen", len(c.index), "idx", idx)
	}

	return nil
}

func (c *Cache) Set(v *VarBind) {
	c.Lock()
	defer c.Unlock()

	slog.Debug("staging", slog.Any("value", v.Marshal()))

	c.staged[v.OID.String()] = v
}
