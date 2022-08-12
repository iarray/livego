package configure

import (
	"fmt"

	"github.com/gwuhaolin/livego/utils/uid"

	"github.com/go-redis/redis/v7"
	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
)

const (
	CACHE_KEY_PREFIX  = "key:"
	CACHE_ROOM_PREFIX = "room:"
)

type RoomKeysType struct {
	redisCli   *redis.Client
	localCache *cache.Cache
}

var RoomKeys = &RoomKeysType{
	localCache: cache.New(cache.NoExpiration, 0),
}

var saveInLocal = true

func Init() {
	saveInLocal = len(Config.GetString("redis_addr")) == 0
	if saveInLocal {
		return
	}

	RoomKeys.redisCli = redis.NewClient(&redis.Options{
		Addr:     Config.GetString("redis_addr"),
		Password: Config.GetString("redis_pwd"),
		DB:       Config.GetInt("redis_db"),
	})
	log.Info("Redis DB: ", Config.GetInt("redis_db"))
	_, err := RoomKeys.redisCli.Ping().Result()
	if err != nil {
		log.Panic("Redis: ", err)
	}

	log.Info("Redis connected")
}

func GetRealCacheKeyName(key string) string {
	return CACHE_KEY_PREFIX + key
}

func GetRealCacheRoomName(room string) string {
	return CACHE_ROOM_PREFIX + room
}

// set/reset a random key for channel
func (r *RoomKeysType) SetKey(channel string) (key string, err error) {
	if !saveInLocal {
		for {
			key = uid.RandStringRunes(48)
			if _, err = r.redisCli.Get(GetRealCacheKeyName(key)).Result(); err == redis.Nil {
				err = r.redisCli.Set(GetRealCacheRoomName(channel), key, 0).Err()
				if err != nil {
					return
				}

				err = r.redisCli.Set(GetRealCacheKeyName(key), channel, 0).Err()
				return
			} else if err != nil {
				return
			}
		}
	}

	for {
		key = uid.RandStringRunes(48)
		if _, found := r.localCache.Get(GetRealCacheKeyName(key)); !found {
			r.localCache.SetDefault(GetRealCacheRoomName(channel), key)
			r.localCache.SetDefault(GetRealCacheKeyName(key), channel)
			break
		}
	}
	return
}

func (r *RoomKeysType) GetKey(channel string) (newKey string, err error) {
	if !saveInLocal {
		if newKey, err = r.redisCli.Get(GetRealCacheRoomName(channel)).Result(); err == redis.Nil {
			newKey, err = r.SetKey(channel) //注意: SetKey 里面已经加了前缀, 这里就别加了
			log.Debugf("[KEY] new channel [%s]: %s", channel, newKey)
			return
		}

		return
	}

	var key interface{}
	var found bool
	if key, found = r.localCache.Get(GetRealCacheRoomName(channel)); found {
		return key.(string), nil
	}
	newKey, err = r.SetKey(channel) //注意: SetKey 里面已经加了前缀, 这里就别加了
	log.Debugf("[KEY] new channel [%s]: %s", channel, newKey)
	return
}

func (r *RoomKeysType) GetChannel(key string) (channel string, err error) {
	if !saveInLocal {
		return r.redisCli.Get(GetRealCacheKeyName(key)).Result()
	}

	chann, found := r.localCache.Get(GetRealCacheKeyName(key))
	if found {
		return chann.(string), nil
	} else {
		return "", fmt.Errorf("%s does not exists", key)
	}
}

func (r *RoomKeysType) DeleteChannel(channel string) bool {
	if !saveInLocal {
		return r.redisCli.Del(GetRealCacheRoomName(channel)).Err() != nil
	}

	key, ok := r.localCache.Get(GetRealCacheRoomName(channel))
	if ok {
		r.localCache.Delete(GetRealCacheRoomName(channel))
		r.localCache.Delete(GetRealCacheKeyName(key.(string)))
		return true
	}
	return false
}

func (r *RoomKeysType) DeleteKey(key string) bool {
	if !saveInLocal {
		return r.redisCli.Del(GetRealCacheKeyName(key)).Err() != nil
	}

	channel, ok := r.localCache.Get(key)
	if ok {
		r.localCache.Delete(GetRealCacheRoomName(channel.(string)))
		r.localCache.Delete(GetRealCacheKeyName(key))
		return true
	}
	return false
}
