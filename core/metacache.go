package core

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2023 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type MetaCache struct {
	maxCacheTime int64
	items        map[int]*MetaCacheItem
}

type MetaCacheItem struct {
	meta *InstanceMeta
	date int64
}

// ////////////////////////////////////////////////////////////////////////////////// //

// NewMetaCache creates new meta cache
func NewMetaCache(maxCacheTime time.Duration) *MetaCache {
	return &MetaCache{
		maxCacheTime: int64(maxCacheTime),
		items:        make(map[int]*MetaCacheItem),
	}
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Set add object to cache
func (c *MetaCache) Set(key int, meta *InstanceMeta) {
	c.items[key] = &MetaCacheItem{meta, time.Now().UnixNano()}
	c.clearCache()
}

// Get returns meta from cache
func (c *MetaCache) Get(key int) (bool, *InstanceMeta) {
	item, hit := c.items[key]

	if hit {
		if time.Now().UnixNano()-item.date >= c.maxCacheTime {
			delete(c.items, key)
			return false, nil
		}

		return true, c.getClone(item.meta)
	}

	return false, nil
}

// Has checks that we have cached meta for instance
func (c *MetaCache) Has(key int) bool {
	item, hit := c.items[key]

	if !hit {
		return false
	}

	if time.Now().UnixNano()-item.date >= c.maxCacheTime {
		delete(c.items, key)
		return false
	}

	return true
}

// Remove removes item from cache
func (c *MetaCache) Remove(key int) {
	delete(c.items, key)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// clearCache removes outdated object
func (c *MetaCache) clearCache() {
	if c.items == nil {
		return
	}

	now := time.Now().UnixNano()

	for key, item := range c.items {
		if now-item.date >= c.maxCacheTime {
			delete(c.items, key)
		}
	}
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
		MetaVersion:     original.MetaVersion,
		ID:              original.ID,
		Desc:            original.Desc,
		ReplicationType: original.ReplicationType,
		UUID:            original.UUID,
		Compatible:      original.Compatible,
		Created:         original.Created,
		Tags:            append([]string(nil), original.Tags...),
		Sentinel:        original.Sentinel,
		Preferencies: &InstancePreferencies{
			ID:             original.Preferencies.ID,
			Password:       original.Preferencies.Password,
			Prefix:         original.Preferencies.Prefix,
			IsSecure:       original.Preferencies.IsSecure,
			IsSaveDisabled: original.Preferencies.IsSaveDisabled,
		},
		AuthInfo: &InstanceAuthInfo{
			Pepper: original.AuthInfo.Pepper,
			Hash:   original.AuthInfo.Hash,
			User:   original.AuthInfo.User,
		},
		ConfigInfo: &InstanceConfigInfo{
			Hash: original.ConfigInfo.Hash,
			Date: original.ConfigInfo.Date,
		},
		Storage: storage,
	}
}
