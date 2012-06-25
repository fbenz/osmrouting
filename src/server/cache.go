// The idea is quite natural, but has been presented by other groups first: cache route requests
// At the moment every request is cached until the cache is full. This can be improved
// by using a hit counter, a timestamp and an eviction policy.

package main

const (
	MaxCacheSize = 100 * 1024 * 1024 // in bytes
	CacheQueueSize = 100
)

var (
	cache Cache
)

type Cache struct {
	Size 	int
	Map 	map[string]*CacheElement
	Queue 	chan *CacheQueueElement
}

type CacheElement struct {
	Response 	[]byte
	HitCounter 	int
}

type CacheQueueElement struct {
	Key			string
	Response 	[]byte
}

func InitCache() {
	cacheMap := make(map[string]*CacheElement)
	cacheQueue := make(chan *CacheQueueElement, CacheQueueSize)
	cache = Cache{Size: 0, Map: cacheMap, Queue: cacheQueue}
	go CacheHandler()
}

func CacheGet(key string) ([]byte, bool) {
	if elem, ok := cache.Map[key]; ok {
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
		if c, ok := cache.Map[elem.Key]; ok {
			c.HitCounter++
		} else if cache.Size + len(elem.Response) <= MaxCacheSize {
			cache.Map[elem.Key] = &CacheElement{Response: elem.Response, HitCounter: 0}
			cache.Size += len(elem.Response)
		}
	}
}
