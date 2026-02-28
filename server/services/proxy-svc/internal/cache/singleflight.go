package cache

import (
	"golang.org/x/sync/singleflight"
)

// SingleFlight 防止缓存击穿（并发请求合并）
type SingleFlight struct {
	group singleflight.Group
}

// NewSingleFlight 创建SingleFlight
func NewSingleFlight() *SingleFlight {
	return &SingleFlight{}
}

// Do 执行函数，相同key的并发请求只执行一次
func (s *SingleFlight) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	v, err, _ := s.group.Do(key, fn)
	return v, err
}

// DoChan 异步执行，返回channel
func (s *SingleFlight) DoChan(key string, fn func() (interface{}, error)) <-chan singleflight.Result {
	return s.group.DoChan(key, fn)
}

// Forget 忘记key，允许下次请求重新执行
func (s *SingleFlight) Forget(key string) {
	s.group.Forget(key)
}
