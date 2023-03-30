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
	key := fmt.Sprintf("usermagnet+%s", strconv.Itoa(int(id)))
	err := RedisClient.ZIncrBy(key, 1, magnet).Err()
	if err != nil {
		fmt.Println("incr user magnet one fail for ", err)
	}
	fmt.Println("add user magnet hot one done")
	// RedisClient.HIncrBy(key, magnet, 1)
}

type RedisZ struct {
	MagnetString string
	Score        float64
}

// TODO 这里没加限制条件，会影响缓存性能
func GetUserMagnetHot(id uint) []RedisZ {
	userMagnetKey := fmt.Sprintf("usermagnet+%s", strconv.Itoa(int(id)))
	fmt.Println(userMagnetKey, "这里是获取所有的usermagnet的key ")
	// userMagnets, err := RedisClient.HGetAll(userMagnetKey).Result()
	userMagnets, err := RedisClient.ZRevRangeWithScores(userMagnetKey, 0, 10).Result()
	if err != nil {
		fmt.Println("get this user hot magnets fail for", id, err)
		return nil
	}
	sortedHot := make([]RedisZ, 0)
	for _, i := range userMagnets {
		tmpHot := RedisZ{
			MagnetString: i.Member.(string),
			Score:        i.Score,
		}
		sortedHot = append(sortedHot, tmpHot)
	}
	return sortedHot
}
