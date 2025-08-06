package redis

import (
	"bytes"
	"encoding/json"
	"github.com/triapex/auth/internal/infra"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

// Redis holds neccessery fields
// connect to redis server
type Redis struct {
	*redis.Client

	projectName string
}

// New returns a new instance of Redis
func New(uri string, timeOut time.Duration, projectName string) (*Redis, error) {
	log.Println("connecting redis: ", uri)
	opt, err := redis.ParseURL(uri)
	if err != nil {
		log.Println("error parsing redis uri: ", err)
		return nil, err
	}

	rdb := redis.NewClient(opt)
	log.Println("connected redis: ", uri)

	return &Redis{
		Client:      rdb,
		projectName: projectName,
	}, nil
}

func (r *Redis) Ping() error {
	return r.Client.Ping().Err()
}

func (r *Redis) prepareKey(tab, key string) string {
	if r.projectName != "" {
		return r.projectName + "_" + tab + "_" + key
	}

	return tab + "_" + key
}

// Get finds a value with the key
func (r *Redis) Get(tab, key string, val interface{}) error {
	data, err := r.Client.Get(r.prepareKey(tab, key)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return infra.ErrNotFound
		}
		return err
	}
	if err := json.Unmarshal(data, val); err != nil {
		return err
	}
	return nil
}

// Put puts a value with the key
func (r *Redis) Put(tab, key string, val interface{}) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return r.Client.Set(r.prepareKey(tab, key), data, 0).Err()
}

// PutEx puts a value with the key for duration d
func (r *Redis) PutEx(tab, key string, val interface{}, d time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return r.Client.Set(r.prepareKey(tab, key), data, d).Err()
}

func (r *Redis) PutNxEx(tab, key string, val interface{}, d time.Duration) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return r.Client.SetNX(r.prepareKey(tab, key), data, d).Err()
}

func (r *Redis) PutNx(tab, key string, val interface{}) error {
	data, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return r.Client.SetNX(r.prepareKey(tab, key), data, 0).Err()
}

// Del deletes a value with the key
func (r *Redis) Del(tab, key string) error {
	return r.Client.Del(r.prepareKey(tab, key)).Err()
}

func (r *Redis) DeleteByPattern(tab, pattern string) error {
	keys, err := r.Client.Keys(r.prepareKey(tab, pattern)).Result()
	if err != nil {
		return err
	}
	if len(keys) > 0 {
		return r.Client.Del(keys...).Err()
	}
	return nil
}

// List lists values with associated keys
func (r *Redis) List(tab string, keys []string, val interface{}) error {
	ks := []string{}
	for _, k := range keys {
		ks = append(ks, r.prepareKey(tab, k))
	}
	res, err := r.Client.MGet(ks...).Result()
	if err != nil {
		return err
	}

	buf := bytes.NewBufferString("[")
	c := 0
	for _, d := range res {
		if d != nil {
			if c != 0 {
				buf.WriteString(",")
			}
			c++
			buf.WriteString(d.(string))
		}
	}
	buf.WriteString("]")
	if err := json.Unmarshal(buf.Bytes(), val); err != nil {
		return err
	}
	return nil
}

// Inc increase the int value of the key by val
func (r *Redis) Inc(tab, key string, val int) error {
	return r.Client.IncrBy(r.prepareKey(tab, key), int64(val)).Err()
}

// SetEx sets expiry of the key to exp
func (r *Redis) SetEx(tab, key string, exp time.Duration) error {
	b, err := r.Client.Expire(r.prepareKey(tab, key), exp).Result()
	if err != nil {
		return err
	}
	if !b {
		return infra.ErrNotFound
	}
	return nil
}

// GetEx returns the expiry of the key
func (r *Redis) GetEx(tab, key string) (time.Duration, error) {
	d, err := r.Client.TTL(r.prepareKey(tab, key)).Result()
	if err != nil {
		return 0, err
	}
	if d < 0 {
		return 0, infra.ErrNotFound
	}
	return d, nil
}

func (r *Redis) AddToSet(tab, key string, members ...infra.Member) error {
	mzs := make([]redis.Z, 0)
	for _, m := range members {
		mzs = append(mzs, redis.Z{
			Score:  float64(m.Score),
			Member: m.Val,
		})
	}
	return r.Client.ZAdd(r.prepareKey(tab, key), mzs...).Err()
}

func (r *Redis) RemoveFromSet(tab, key string, members ...string) error {
	if err := r.Client.ZRem(r.prepareKey(tab, key), members).Err(); err != nil {
		log.Println("Failed to remove from set", err)
	}
	return nil
}

func (r *Redis) RemoveFromAllSet(tab string, members ...string) error {
	keys, err := r.Client.Keys(r.prepareKey(tab, "*")).Result()
	if err != nil {
		return err
	}
	wg := sync.WaitGroup{}
	for _, k := range keys {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			//log.Println(key, ">>>>", members)
			if err := r.Client.ZRem(key, members).Err(); err != nil {
				log.Println("Failed to remove from set", err)
			}
		}(k)
	}
	wg.Wait()
	return nil
}
func (r *Redis) RemoveFromAllSetByKeys(tab string, keys []string, members ...string) error {
	wg := sync.WaitGroup{}
	for _, k := range keys {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			log.Println(key, ">>>>", members)
			if err := r.Client.ZRem(r.prepareKey(tab, key), members).Err(); err != nil {
				log.Println("Failed to remove from set", err)
			}
		}(k)
	}
	wg.Wait()
	return nil
}

func (r *Redis) ListAllKeys(tab string) ([]string, error) {
	keys, err := r.Client.Keys(r.prepareKey(tab, "*")).Result()
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *Redis) ListMemberFromSet(tab, key string, skip, limit int64) ([]string, error) {

	return r.Client.ZRange(r.prepareKey(tab, key), skip, skip+limit).Result()
}

func (r *Redis) ListDataFromSet(tab, key string, skip, limit int64, data interface{}, asc bool) error {
	tabKey := r.prepareKey(tab, key)
	stop := skip + limit - 1
	start := skip
	var members []string
	var err error
	if asc {
		members, err = r.Client.ZRange(tabKey, start, stop).Result()
	} else {
		members, err = r.Client.ZRevRange(tabKey, start, stop).Result()
	}
	if err != nil {
		log.Println("Failed to list from set", err)
		return err
	}
	if len(members) > 0 && len(members) == int(limit) {
		return r.List(tab, members, data)
	}
	return nil
}

func (r *Redis) ListDataFromSetWithRetrival(listTab, retrivalTab, key string, skip, limit int64, data interface{}, asc bool) error {
	tabKey := r.prepareKey(listTab, key)
	stop := skip + limit - 1
	start := skip
	var members []string
	var err error
	if asc {
		members, err = r.Client.ZRange(tabKey, start, stop).Result()
	} else {
		members, err = r.Client.ZRevRange(tabKey, start, stop).Result()
	}
	if err != nil {
		log.Println("Failed to list from set", err)
		return err
	}
	log.Println(members, ">>>>>")
	if len(members) > 0 && len(members) == int(limit) {
		return r.List(retrivalTab, members, data)
	}
	return nil
}
