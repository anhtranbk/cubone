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

func NewInmemoryBloomFilter() *InmemoryBloomFilter {
	return &InmemoryBloomFilter{
		db: make(map[string]byte),
	}
}

//InmemoryBloomFilter for development purpose
type InmemoryBloomFilter struct {
	db map[string]byte
}

func (i InmemoryBloomFilter) Exists(key string, item string) (bool, error) {
	_, found := i.db[key + item]
	return found, nil
}

func (i InmemoryBloomFilter) Add(key string, item string) error {
	i.db[key + item] = 1
	return nil
}

func (r RedisBloomFilter) Exists(key string, item string) (bool, error) {
	panic("implement me")
}

func (r RedisBloomFilter) Add(key string, item string) error {
	panic("implement me")
}

