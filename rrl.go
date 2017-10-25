package rrl

import (
	"crypto/rand"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/go-redis/redis"
	"errors"
)


const concurrent_requests_limiter_lua = `
local key = KEYS[1]

local capacity = tonumber(ARGV[1])
local timestamp = tonumber(ARGV[2])
local id = ARGV[3]
local clearBefore = tonumber(ARGV[4])
local newZSetExpireTime = tonumber(ARGV[5])


redis.call("zremrangebyscore", key, 0, clearBefore)
redis.call("expire", key, newZSetExpireTime)

local count = redis.call("zcard", key)
local allowed = count < capacity

if allowed then
	redis.call("zadd", key, timestamp, id)
end

return { allowed, count }
`

func uuid() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

//RateLimiter implements a rolling rate limiter using a Redis zset and atomic
//batching of operations.
type RateLimiter struct {
	intervalInMillis int64
	maxInInterval    int
	client           *redis.Client
}

//NewRateLimiter implements a rate limiter instance, which can calculate how long until
//the next request will be accepted given the length of the rolling interval and how
//many requests are allowed in an interval.
func NewRateLimiter(intervalInMillis int64, maxInInterval int, client *redis.Client) *RateLimiter {
	return &RateLimiter{
		intervalInMillis: intervalInMillis,
		maxInInterval:    maxInInterval,
		client:           client,
	}
}

//AllowRequest true if the request for the given id falls within
//the max allowed requests for the time interval, false otherwise.
func (rl *RateLimiter) AllowRequest(id string) (bool, error) {
	now := time.Now().UnixNano() / 1000               //microseconds
	clearBefore := now - (rl.intervalInMillis * 1000) //microseconds
	log.Debug("clearBefore ", clearBefore)

	element, err := uuid()
	if err != nil {
		return false, err
	}

	log.Debug("new element ", element)

	newZSetExpireTime := rl.intervalInMillis/1000
	
	cmd := rl.client.Eval(concurrent_requests_limiter_lua, []string{id},rl.maxInInterval, now,element, clearBefore, newZSetExpireTime)
	if cmd.Err() != nil {
		log.Warn("script execution error", cmd.Err().Error())
		return false, cmd.Err()
	}

	cmdOutput := cmd.Val()
	log.Debug("script output ", cmdOutput)
	outputSlice, ok := cmdOutput.([]interface{})
	if !ok {
		return false, errors.New("Unexcepted result type from Redis script execution")
	}

	return outputSlice[0] != nil, nil
}

