package cache

import "github.com/go-redis/redis"

var RedisClient *redis.Client

func Redis() {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err := client.Ping().Result()
	if err != nil {
		panic("redis init failed")
	}
	RedisClient = client
}
