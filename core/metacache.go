package core

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"sync"
	"time"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// MetaCache is instance metadata cache
type MetaCache struct {
	maxCacheTime int64
	items        *sync.Map
}

// MetaCacheItem contains instance metadata and date of creation
type MetaCacheItem struct {
	meta *InstanceMeta
	date int64
}

// ////////////////////////////////////////////////////////////////////////////////// //

// NewMetaCache creates new meta cache
func NewMetaCache(maxCacheTime time.Duration) *MetaCache {
	return &MetaCache{
		maxCacheTime: int64(maxCacheTime),
		items:        &sync.Map{},
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Set adds metadata to cache
func (c *MetaCache) Set(key int, meta *InstanceMeta) {
	if c.items == nil {
		return
	}

	c.items.Store(key, &MetaCacheItem{meta, time.Now().UnixNano()})

	c.clearCache()
}

// Get returns meta from cache if exist
func (c *MetaCache) Get(key int) (*InstanceMeta, bool) {
	if c.items == nil {
		return nil, false
	}

	v, ok := c.items.Load(key)

	if ok {
		item := v.(*MetaCacheItem)

		if time.Now().UnixNano()-item.date >= c.maxCacheTime {
			c.items.Delete(key)
			return nil, false
		}

		return c.getClone(item.meta), true
	}

	return nil, false
}

// Has checks that we have cached meta for instance
func (c *MetaCache) Has(key int) bool {
	if c.items == nil {
		return false
	}

	v, ok := c.items.Load(key)

	if !ok {
		return false
	}

	item := v.(*MetaCacheItem)

	if time.Now().UnixNano()-item.date >= c.maxCacheTime {
		c.items.Delete(key)
		return false
	}

	return true
}

// Remove removes item from cache
func (c *MetaCache) Remove(key int) {
	if c.items == nil {
		return
	}

	c.items.Delete(key)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// clearCache removes outdated object
func (c *MetaCache) clearCache() {
	if c.items == nil {
		return
	}

	now := time.Now().UnixNano()

	c.items.Range(func(key, value any) bool {
		item := value.(*MetaCacheItem)

		if now-item.date >= c.maxCacheTime {
			c.items.Delete(key)
		}

		return true
	})
}

// getClone clones meta
func (c *MetaCache) getClone(original *InstanceMeta) *InstanceMeta {
	var storage map[string]string

	if original.Storage != nil {
		storage = make(map[string]string)

		for k, v := range original.Storage {
			storage[k] = v
		}
	}

	return &InstanceMeta{
		MetaVersion: original.MetaVersion,
		ID:          original.ID,
		Desc:        original.Desc,
		UUID:        original.UUID,
		Compatible:  original.Compatible,
		Created:     original.Created,
		Tags:        append([]string(nil), original.Tags...),
		Preferencies: &InstancePreferencies{
			AdminPassword:    original.Preferencies.AdminPassword,
			SyncPassword:     original.Preferencies.SyncPassword,
			ServicePassword:  original.Preferencies.ServicePassword,
			SentinelPassword: original.Preferencies.SentinelPassword,
			ReplicationType:  original.Preferencies.ReplicationType,
			IsSaveDisabled:   original.Preferencies.IsSaveDisabled,
		},
		Auth: &InstanceAuth{
			Pepper: original.Auth.Pepper,
			Hash:   original.Auth.Hash,
			User:   original.Auth.User,
		},
		Config: &InstanceConfigInfo{
			Hash: original.Config.Hash,
			Date: original.Config.Date,
		},
		Storage: storage,
	}
}
