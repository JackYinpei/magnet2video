package cache

import (
	"fmt"
	"strconv"
)

const (
	HotRankKey = "rank:hot"
)

func MagnetViewKey(id uint) string {
	return fmt.Sprintf("view:magnet:%s", strconv.Itoa(int(id)))
}

func UserMagnetViewCountPlusOne(id uint, magnet string) {
	key := fmt.Sprintf("usermagnet:%s", strconv.Itoa(int(id)))
	RedisClient.HIncrBy(key, magnet, 1)
}

func GetUserMagnetHot(id uint, magnet string) []string {
	userMagnetKey := fmt.Sprintf("usermagnet:%s", strconv.Itoa(int(id)))
	userMagnets, err := RedisClient.HGetAll(userMagnetKey).Result()
	if err != nil {
		fmt.Println("get this user hot magnets fail for", id, err)
		return nil
	}
	// TODO 排序key value
	for k, v := range userMagnets {

	}
}
