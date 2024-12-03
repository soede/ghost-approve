package redis

import (
	"fmt"
	"github.com/redis/go-redis/v9"
)

func Client() *Store {
	return rdb
}

func (r *Store) SetValue(key string, value string) error {
	err := r.client.Set(r.ctx, key, value, 0).Err()
	if err != nil {
		return fmt.Errorf("ошибка установки значения: %w", err)
	}
	return nil
}

func (r *Store) GetValue(key string) (string, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return "", fmt.Errorf("ошибка получения значения: %w", err)
	}
	return val, nil
}

func (r *Store) ZAdd(key string, member string, score float64) error {
	add := r.client.ZAdd(r.ctx, key, redis.Z{
		Score:  score,
		Member: member,
	})

	return add.Err()
}

func (r *Store) ZRem(key string, taskMember string) error {
	remove := r.client.ZRem(r.ctx, key, taskMember)

	return remove.Err()
}

func (r *Store) ZScore(key string, taskMember string) (float64, error) {
	result, err := r.client.ZScore(r.ctx, key, taskMember).Result()

	return result, err
}

func (r *Store) ZRangeByScoreWithScores(key string, min string, max string) ([]redis.Z, error) {
	tasks, err := r.client.ZRangeByScoreWithScores(r.ctx, key, &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения задач из Redis: %w", err)
	}

	return tasks, nil

}
