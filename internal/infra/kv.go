package infra

import (
	"time"
)

// KV interface represents the key value storage
type KV interface {
	Ping() error
	Get(tab, key string, val interface{}) error
	List(tab string, keys []string, val interface{}) error
	Put(tab, key string, val interface{}) error
	PutEx(tab, key string, val interface{}, d time.Duration) error
	PutNxEx(tab, key string, val interface{}, d time.Duration) error
	PutNx(tab, key string, val interface{}) error
	Inc(tab, key string, val int) error
	SetEx(tab, key string, exp time.Duration) error
	GetEx(tab, key string) (time.Duration, error)
	Del(tab, key string) error
	DeleteByPattern(tab, pattern string) error
	AddToSet(tab, key string, members ...Member) error
	RemoveFromSet(tab, key string, members ...string) error
	RemoveFromAllSet(tab string, members ...string) error
	RemoveFromAllSetByKeys(tab string, keys []string, members ...string) error
	ListDataFromSet(tab, key string, skip, limit int64, data interface{}, asc bool) error
	ListDataFromSetWithRetrival(listTab, retrivalTab, key string, skip, limit int64, data interface{}, asc bool) error
}
type Member struct {
	Score int64
	Val   interface{}
}
