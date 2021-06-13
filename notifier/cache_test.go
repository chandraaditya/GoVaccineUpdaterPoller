package notifier

import (
	"GoVaccineUpdaterPoller/parser"
	"context"
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	rcache "github.com/go-redis/cache/v8"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"reflect"
	"testing"
	"time"
)

func TestInMemoryCache_Contains(t *testing.T) {
	type fields struct {
		cache map[string]notified
	}
	type args struct {
		dose      int
		sessionId string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "non-existant",
			fields: fields{cache: map[string]notified{}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
			want: false,
		},
		{
			name: "existant",
			fields: fields{cache: map[string]notified{
				"poller/1/abcd": {},
			}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := InMemoryCache{
				cache: tt.fields.cache,
			}
			if got := i.Contains(tt.args.dose, tt.args.sessionId); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemoryCache_Get(t *testing.T) {
	type fields struct {
		cache map[string]notified
	}
	type args struct {
		dose      int
		sessionId string
	}
	now := time.Now()
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantSession  parser.Session
		wantCaughtAt time.Time
	}{
		{
			name:   "non-existant",
			fields: fields{cache: map[string]notified{}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
		},
		{
			name: "existant",
			fields: fields{cache: map[string]notified{
				"poller/1/abcd": {
					Session: parser.Session{
						SessionId: "abcd",
					},
					TimeCaught: now,
				},
			}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
			wantSession: parser.Session{
				SessionId: "abcd",
			},
			wantCaughtAt: now,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := InMemoryCache{
				cache: tt.fields.cache,
			}
			gotSession, gotCaughtAt := i.Get(tt.args.dose, tt.args.sessionId)
			if !reflect.DeepEqual(gotSession, tt.wantSession) {
				t.Errorf("Get() gotSession = %v, want %v", gotSession, tt.wantSession)
			}
			if !reflect.DeepEqual(gotCaughtAt, tt.wantCaughtAt) {
				t.Errorf("Get() gotCaughtAt = %v, want %v", gotCaughtAt, tt.wantCaughtAt)
			}
		})
	}
}

func TestInMemoryCache_Put(t *testing.T) {
	now := time.Now()
	type fields struct {
		cache map[string]notified
	}
	type args struct {
		dose    int
		session parser.Session
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "put",
			fields: fields{cache: map[string]notified{
				"poller/1/abcd": {
					Session: parser.Session{
						SessionId: "abcd",
					},
					TimeCaught: now,
				},
			}},
			args: args{
				dose: 1,
				session: parser.Session{
					SessionId: "abcd",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := map[string]notified{}
			i := InMemoryCache{
				cache: cache,
			}
			i.Put(tt.args.dose, tt.args.session)
			require.Contains(t, cache, "poller/1/abcd")
			require.Equal(t, tt.fields.cache["poller/1/abcd"].Session, cache["poller/1/abcd"].Session)
		})
	}
}

func TestInMemoryCache_Remove(t *testing.T) {
	type fields struct {
		cache map[string]notified
	}
	type args struct {
		dose      int
		sessionId string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "remove",
			fields: fields{cache: map[string]notified{
				"poller/1/abcd": {
					Session: parser.Session{
						SessionId: "abcd",
					},
				},
			}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := InMemoryCache{
				cache: tt.fields.cache,
			}
			i.Remove(1, "abcd")
			require.NotContains(t, i.cache, "poller/1/abcd")
			require.Equal(t, 0, len(i.cache))
		})
	}
}

var testLog logr.Logger

func globalSetup(t *testing.T) {
	testLog = zapr.NewLogger(zaptest.NewLogger(t))
}

func TestRedisCache_Put(t *testing.T) {
	globalSetup(t)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	c := rcache.New(&rcache.Options{
		Redis: client,
	})
	redisCache := RedisCache{
		log:        testLog,
		expiration: time.Hour,
		cache:      c,
	}
	now := time.Now()
	type fields struct {
		cache map[string]notified
	}
	type args struct {
		dose    int
		session parser.Session
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "put",
			fields: fields{cache: map[string]notified{
				"poller/1/abcd": {
					Session: parser.Session{
						SessionId: "abcd",
					},
					TimeCaught: now,
				},
			}},
			args: args{
				dose: 1,
				session: parser.Session{
					SessionId: "abcd",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redisCache.Put(tt.args.dose, tt.args.session)
			val, err := client.Get(ctx, "poller/1/abcd").Result()
			require.NoError(t, err)
			require.Greater(t, len(val), 5)
		})
	}
}

func TestRedisCache_Get(t *testing.T) {
	globalSetup(t)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	c := rcache.New(&rcache.Options{
		Redis: client,
	})
	redisCache := RedisCache{
		log:        testLog,
		expiration: time.Hour,
		cache:      c,
	}
	type fields struct {
		cache map[string]notified
	}
	type args struct {
		dose      int
		sessionId string
	}
	now := time.Now()
	tests := []struct {
		name         string
		fields       fields
		args         args
		wantSession  parser.Session
		wantCaughtAt time.Time
	}{
		{
			name:   "non-existant",
			fields: fields{cache: map[string]notified{}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
		},
		{
			name: "existant",
			fields: fields{cache: map[string]notified{
				"poller/1/abcd": {
					Session: parser.Session{
						SessionId: "abcd",
					},
					TimeCaught: now,
				},
			}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
			wantSession: parser.Session{
				SessionId: "abcd",
			},
			wantCaughtAt: now,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = client.FlushDB(context.Background()).Result()
			for key, value := range tt.fields.cache {
				err := c.Set(&rcache.Item{
					Ctx:   ctx,
					Key:   key,
					Value: value,
				})
				require.NoError(t, err)
			}
			gotSession, gotCaughtAt := redisCache.Get(tt.args.dose, tt.args.sessionId)
			if !reflect.DeepEqual(gotSession, tt.wantSession) {
				t.Errorf("Get() gotSession = %v, want %v", gotSession, tt.wantSession)
			}
			require.True(t, tt.wantCaughtAt.Equal(gotCaughtAt))
		})
	}
}

func TestRedisCache_Contains(t *testing.T) {
	globalSetup(t)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	c := rcache.New(&rcache.Options{
		Redis: client,
	})
	redisCache := RedisCache{
		log:        testLog,
		expiration: time.Hour,
		cache:      c,
	}
	type fields struct {
		cache map[string]notified
	}
	type args struct {
		dose      int
		sessionId string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "non-existant",
			fields: fields{cache: map[string]notified{}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
			want: false,
		},
		{
			name: "existant",
			fields: fields{cache: map[string]notified{
				"poller/1/abcd": {},
			}},
			args: args{
				dose:      1,
				sessionId: "abcd",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _ = client.FlushDB(context.Background()).Result()
			for key, value := range tt.fields.cache {
				err := c.Set(&rcache.Item{
					Ctx:   ctx,
					Key:   key,
					Value: value,
				})
				require.NoError(t, err)
			}
			if got := redisCache.Contains(tt.args.dose, tt.args.sessionId); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
