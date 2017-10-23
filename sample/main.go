package main

import (
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/xtraclabs/rrl"
)

const maxRequestsPerSecond = 20

func SampleHandler(w http.ResponseWriter, r *http.Request) {
	log.Info("Sample handler called")
}

func RateLimitedHandler(redisClient *redis.Client, h func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	governor := rrl.NewRateLimiter(60*1000, 60*maxRequestsPerSecond, redisClient)
	return func(w http.ResponseWriter, r *http.Request) {
		//Here we are assuming everyone is app 1 - we could pull this from the JWT to apply different
		//policies per application.

		timeleft, err := governor.TimeLeft("a1")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if timeleft != 0 {
			http.Error(w, fmt.Sprintf("Retry request in %d milliseconds", timeleft), http.StatusTooManyRequests)
			return
		}

		//Not throttled - proceed.
		h(w, r)
	}
}

func main() {
	log.SetLevel(log.DebugLevel)

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	r := mux.NewRouter()
	r.HandleFunc("/foo", RateLimitedHandler(client, SampleHandler))
	http.Handle("/", r)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
