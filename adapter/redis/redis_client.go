/*
 @Title
 @Description
 @Author  Leo
 @Update  2020/8/6 2:20 下午
*/

package redis

import (
	"encoding/json"

	"github.com/gomodule/redigo/redis"
)

// 已完成指令
// 字符串 set / get
// 哈希 hset / hget / hmset / hmget / hgetall
// 列表 lpush / rpop / brpop

type RedisClient struct {
	Client *redis.Pool
}

func (rc *RedisClient) Get() redis.Conn {
	return rc.Client.Get()
}

// 阻塞从右侧弹出一个队列元素（支持同时监控多个队列）
// 注意：超时时间设置需要参考 redis 服务器 keep alive 设置，如果超过了该值，会由于服务端主动断开触发 timeout 错误
// 自行判断 redis.ErrNil redis 空值 error
func (rc *RedisClient) Brpop(keys []string, blockFor int64) (k, result string, e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	r, e := redis.Strings(client.Do("BRPOP", redis.Args{}.AddFlat(keys).Add(blockFor)...))
	if e != nil {
		return "", "", e
	}

	return r[0], r[1], nil
}

// 从右侧弹出一个队列元素
func (rc *RedisClient) Rpop(key string) (result string, e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	result, e = redis.String(client.Do("RPOP", key))
	return result, e
}

// 从左侧压入一批元素到队列
func (rc *RedisClient) LBatchPush(key string, value []string) (e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	_, e = client.Do("LPUSH", redis.Args{}.Add(key).AddFlat(value)...)
	return e
}

// 从左侧压入一个元素到队列
func (rc *RedisClient) Lpush(key, value string) (e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	_, e = client.Do("LPUSH", key, value)
	return e
}

// 获取一个 hash 结构的全部值
func (rc *RedisClient) Hgetall(key string) (result map[string]string, e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	result = make(map[string]string)

	r, e := redis.Strings(client.Do("HGETALL", key))

	for i, c := 0, len(r); i < c; i += 2 {
		result[r[i]] = r[i+1]
	}

	return result, e
}

// 批量获取一个 hash 结构的多个字段值
func (rc *RedisClient) Hmget(key string, fields []string) (result map[string]string, e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	result = make(map[string]string)

	r, e := redis.Strings(client.Do("HMGET", redis.Args{}.Add(key).AddFlat(fields)...))

	for i, v := range r {
		result[fields[i]] = v
	}

	return result, e
}

// 对一个 hash 结构批量设置键值对
func (rc *RedisClient) Hmset(key string, kvs map[string]string) (e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	_, e = client.Do("HMSET", redis.Args{}.Add(key).AddFlat(kvs)...)
	return e
}

// 获取一个 hash 结构单一字段值
func (rc *RedisClient) Hget(key, field string) (result string, e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	result, e = redis.String(client.Do("HGET", key, field))
	return result, e
}

// 对一个 hash 结构设置单一键值对
func (rc *RedisClient) Hset(key, field, value string) (e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	_, e = client.Do("HSET", key, field, value)
	return e
}

// get 命令，从 redis 获取一个字符串
func (rc *RedisClient) GetString(key string) (str string, e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	_d, e := redis.String(client.Do("GET", key))
	if e != nil {
		//if e==redis.ErrNil {
		//	return "",nil // 交给外部了
		//}
		return "", e
	}

	return _d, nil
}

// 同 set 命令，设置一个字符串到 redis
func (rc *RedisClient) SetString(key, value string, expire int64) (e error) {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	if expire > 0 {
		_, e = client.Do("SETEX", key, expire, value)
	} else {
		_, e = client.Do("SET", key, value)
	}

	return e
}

// 从 redis 获取结构体,并解析到对应结构体指针
// data 新建的结构体指针，数据将反 json 化到该内存
func (rc *RedisClient) GetStruct(key string, data interface{}) error {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	_d, e := redis.Bytes(client.Do("GET", key))
	if e != nil {
		return e
	}

	e = json.Unmarshal(_d, data)
	if e != nil {
		return e
	}

	return nil
}

// 保存结构体到 redis
// data : 结构体指针
// 自动 json 化结构体并保存到 redis
func (rc *RedisClient) SetStruct(key string, data interface{}, expire int64) error {
	client := rc.Get()
	defer func() {
		_ = client.Close()
	}()

	d, e := json.Marshal(data)
	if e != nil {
		return e
	}

	if expire > 0 {
		_, e = client.Do("SETEX", key, expire, d)
	} else {
		_, e = client.Do("SET", key, d)
	}

	if e != nil {
		return e
	}

	return nil
}
