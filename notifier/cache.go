package notifier

import (
	"GoVaccineUpdaterPoller/parser"
	"context"
	"fmt"
	"github.com/go-logr/logr"
	rcache "github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"time"
)

type notified struct {
	Session    parser.Session
	TimeCaught time.Time
}

type NotificationCache interface {
	Get(dose int, sessionId string) (session parser.Session, caughtAt time.Time)
	Put(dose int, session parser.Session)
	Contains(dose int, sessionId string) bool
	Remove(dose int, sessionId string)
}

func NewCache(cacheType string, log logr.Logger, redisHost string, redisPassword string, dbIndex int, ttl time.Duration) *NotificationCache {
	var cache NotificationCache
	switch cacheType {
	case "in-memory":
		cache = new(InMemoryCache)
		cache.(*InMemoryCache).cache = map[string]notified{}
	case "redis":
		cache = new(RedisCache)
		client := redis.NewClient(&redis.Options{
			Addr:     redisHost,
			Password: redisPassword,
			DB:       dbIndex,
		})
		c := rcache.New(&rcache.Options{
			Redis: client,
		})
		cache.(*RedisCache).cache = c
		cache.(*RedisCache).log = log.WithName("redis")
		cache.(*RedisCache).expiration = ttl
	default:
		panic("unknown cache type")
	}
	return &cache
}

type InMemoryCache struct {
	cache map[string]notified
}

func (i InMemoryCache) Get(dose int, sessionId string) (session parser.Session, caughtAt time.Time) {
	key := fmt.Sprintf("poller/%d/%s", dose, sessionId)
	if value, ok := i.cache[key]; ok {
		caughtAt = value.TimeCaught
		session = value.Session
	}
	return
}

func (i InMemoryCache) Put(dose int, session parser.Session) {
	key := fmt.Sprintf("poller/%d/%s", dose, session.SessionId)
	i.cache[key] = notified{
		Session:    session,
		TimeCaught: time.Now(),
	}
}

func (i InMemoryCache) Contains(dose int, sessionId string) bool {
	key := fmt.Sprintf("poller/%d/%s", dose, sessionId)
	_, ok := i.cache[key]
	return ok
}

func (i InMemoryCache) Remove(dose int, sessionId string) {
	key := fmt.Sprintf("poller/%d/%s", dose, sessionId)
	delete(i.cache, key)
}

var ctx = context.Background()

type RedisCache struct {
	log        logr.Logger
	expiration time.Duration
	cache      *rcache.Cache
}

func (r RedisCache) Get(dose int, sessionId string) (session parser.Session, caughtAt time.Time) {
	key := fmt.Sprintf("poller/%d/%s", dose, sessionId)
	var result notified
	err := r.cache.Get(ctx, key, &result)
	if err == nil {
		session = result.Session
		caughtAt = result.TimeCaught
	} else {
		r.log.Error(err, err.Error())
	}
	return
}

func (r RedisCache) Put(dose int, session parser.Session) {
	key := fmt.Sprintf("poller/%d/%s", dose, session.SessionId)
	value := notified{
		Session:    session,
		TimeCaught: time.Now(),
	}
	if err := r.cache.Set(&rcache.Item{
		Ctx:   ctx,
		Key:   key,
		Value: value,
		TTL:   r.expiration,
	}); err != nil {
		r.log.Error(err, err.Error())
	}
}

func (r RedisCache) Contains(dose int, sessionId string) bool {
	key := fmt.Sprintf("poller/%d/%s", dose, sessionId)
	return r.cache.Exists(ctx, key)
}

func (r RedisCache) Remove(dose int, sessionId string) {
	key := fmt.Sprintf("poller/%d/%s", dose, sessionId)
	if err := r.cache.Delete(ctx, key); err != nil {
		r.log.Error(err, err.Error())
	}
}
