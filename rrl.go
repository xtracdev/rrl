package rrl

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/go-redis/redis"
)

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

//TimeLeft returns the number of milli-seconds until the next request should be
//made. Zero is returned if there are remaining requests availale within the parameters
//of the rate limiter. For implementing rate limiting, if zero is returned, the
//request should be allowed, other wise it should be rejected.
func (rl *RateLimiter) TimeLeft(id string) (int, error) {
	now := time.Now().UnixNano() / 1000               //microseconds
	clearBefore := now - (rl.intervalInMillis * 1000) //microseconds
	log.Debug("clearBefore ", clearBefore)

	element, err := uuid()
	if err != nil {
		return -1, err
	}

	log.Debug("new element ", element)

	pipeline := rl.client.TxPipeline()
	pipeline.ZRemRangeByScore(id, "0", fmt.Sprintf("%d", clearBefore))
	pipeline.ZRangeWithScores(id, 0, -1)
	pipeline.ZAdd(id, redis.Z{Score: float64(now), Member: element})
	pipeline.Expire(id, time.Duration(rl.intervalInMillis/1000)*time.Second)

	var cmdErr []redis.Cmder
	var pipelineErr error
	cmdErr, pipelineErr = pipeline.Exec()
	if pipelineErr != nil {
		return -1, pipelineErr
	}

	log.Debug("z rem range result ", cmdErr[0])

	rangeWithScoresResult := cmdErr[1]

	elements := zparts(rangeWithScoresResult.String())
	//log.Debug(elements)
	log.Debugf("zset has %d elements", 1+len(elements))

	//Ok to keep making requests?
	if len(elements) < rl.maxInInterval {
		return 0, nil
	}

	//Since the max requests for the interval have been made, the next time a request
	//can be made is when a slot opens up in the zparts. That will the difference between
	//the timestamp of the oldest element and the length of the interval.
	timeleft := (int64(elements[0].Score) - clearBefore) / 1000 //divide by 1000 to return time left in ms

	log.Debugf("timeleft %d", timeleft)

	return int(timeleft), nil
}

//Parse the output of the range with scores command in redis. The output looks like
//[{1.508595813639e+12 1a2942d7-f536-4b6b-a312-88c1465c18c5} {1.508595815287e+12 6bc8a7a7-1435-4a33-8b7d-32edabedd61b}]
func zparts(zstring string) []redis.Z {
	var elements []redis.Z

	parts := strings.Split(zstring, ":")

	if len(parts) < 2 {
		return elements
	}

	zslice := strings.TrimSpace(parts[1])
	if zslice == "[]" {
		return elements
	}

	zslice = strings.Trim(zslice, "[]")
	zpairs := strings.Split(zslice, " ")

	for i := 0; i < len(zpairs); i += 2 {
		rawScore := zpairs[i]
		rawElement := zpairs[i+1]

		score, _ := strconv.ParseFloat(strings.Trim(rawScore, "{"), 64)
		element := strings.Trim(rawElement, "}")

		z := redis.Z{
			Score:  score,
			Member: element,
		}

		elements = append(elements, z)
	}

	return elements
}
