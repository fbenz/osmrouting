/*
 * Copyright 2014 Florian Benz, Steven Schäfer, Bernhard Schommer
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// The idea is quite natural, but has been presented by other groups first: cache route requests

package main

import (
	"sync"
	"time"
)

const (
	MaxCacheSize        = 100 * 1024 * 1024 // in bytes
	CacheQueueSize      = 100
	CacheEvictionBorder = 0.75    // % of cache size
	CacheEvictionPeriod = 15 * 60 // in seconds; min. distance in time between eviction passes
)

var (
	cache Cache
)

type Cache struct {
	Size             int
	Queue            chan *CacheQueueElement
	mutex            sync.Mutex
	elements         map[string]*CacheElement
	lastEvictionPass time.Time
}

type CacheElement struct {
	Response     []byte
	HitCounter   int
	LastAccessed time.Time
}

type CacheQueueElement struct {
	Key      string
	Response []byte
}

// Thread-safe access to the map of the cache
func (c Cache) Get(key string) (*CacheElement, bool) {
	c.mutex.Lock()
	element, ok := c.elements[key]
	c.mutex.Unlock()
	return element, ok
}

// Thread-safe access to the map of the cache
func (c Cache) Put(key string, element *CacheElement) {
	c.mutex.Lock()
	c.elements[key] = element
	c.mutex.Unlock()
}

// Thread-safe access to the map of the cache
func (c Cache) Update(key string) (*CacheElement, bool) {
	c.mutex.Lock()
	element, ok := c.elements[key]
	if ok {
		// adjust element in case of a hit
		element.HitCounter++
		element.LastAccessed = time.Now()
	}
	c.mutex.Unlock()
	return element, ok
}

// Thread-safe access to the map of the cache
func (c Cache) Delete(key string) {
	c.mutex.Lock()
	delete(c.elements, key)
	c.mutex.Unlock()
}

func InitCache() {
	cacheMap := make(map[string]*CacheElement)
	cacheQueue := make(chan *CacheQueueElement, CacheQueueSize)
	cache = Cache{Size: 0, elements: cacheMap, Queue: cacheQueue, lastEvictionPass: time.Now()}
	go CacheHandler()
}

func CacheGet(key string) ([]byte, bool) {
	if elem, ok := cache.Get(key); ok {
		return elem.Response, true
	}
	return nil, false
}

func CachePut(key string, response []byte) {
	if len(cache.Queue) < CacheQueueSize {
		cache.Queue <- &CacheQueueElement{Key: key, Response: response}
	}
}

func CacheHandler() {
	for elem := range cache.Queue {
		if _, ok := cache.Update(elem.Key); !ok && cache.Size+len(elem.Response) <= MaxCacheSize {
			cache.Put(elem.Key, &CacheElement{Response: elem.Response, HitCounter: 0, LastAccessed: time.Now()})
			cache.Size += len(elem.Response)
		}
		if cache.Size >= CacheEvictionBorder*MaxCacheSize {
			timeSinceLastEviction := time.Now().Sub(cache.lastEvictionPass)
			// respect the period
			if timeSinceLastEviction.Seconds() < CacheEvictionPeriod {
				return
			}
			for k, elem := range cache.elements {
				// TODO sort by some criteria and then delete the tail
				timeSinceLastAccess := time.Now().Sub(elem.LastAccessed)
				if elem.HitCounter <= 2 && timeSinceLastAccess.Hours() >= 2 {
					cache.Delete(k)
				}
			}
			cache.lastEvictionPass = time.Now()
		}
	}
}
