package cubone

type BloomFilter interface {
	Exists(key string, item string) (bool, error)
	Add(key string, item string) error
}

func NewRedisBloomFilter(config Config) *RedisBloomFilter {
	return &RedisBloomFilter{}
}

type RedisBloomFilter struct {

}



