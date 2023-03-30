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

// TODO 这里没加限制条件，会影响缓存性能
func GetUserMagnetHot(id uint) map[string]string {
	userMagnetKey := fmt.Sprintf("usermagnet:%s", strconv.Itoa(int(id)))
	fmt.Println(userMagnetKey, "这里是获取所有的usermagnet的key ")
	userMagnets, err := RedisClient.HGetAll(userMagnetKey).Result()
	if err != nil {
		fmt.Println("get this user hot magnets fail for", id, err)
		return nil
	}
	return userMagnets
}
