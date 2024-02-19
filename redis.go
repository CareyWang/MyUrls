package main

import (
	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

// initRedisClient is a function that takes a pointer to a RedisOptions struct and returns a pointer to a Redis client.
func initRedisClient(options *redis.Options) {
	RedisClient = redis.NewClient(options)
}

func GetRedisClient() *redis.Client {
	return RedisClient
}
