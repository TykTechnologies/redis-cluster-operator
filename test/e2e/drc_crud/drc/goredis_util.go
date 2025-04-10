package drc

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/sync/errgroup"
)

const defaultTimeOut = time.Second * 30

// GoRedis contains ClusterClient.
type GoRedis struct {
	client   *redis.ClusterClient
	password string
}

// NewGoRedis return a new ClusterClient.
func NewGoRedis(addr, password string) *GoRedis {
	return &GoRedis{
		client: redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:    []string{addr},
			Password: password,
			//MaxRetries: 5,
			//
			//PoolSize:     3,
			//MinIdleConns: 1,
			//PoolTimeout:  defaultTimeOut,
			//IdleTimeout:  defaultTimeOut,
		}),
		password: password,
	}
}

// StuffingData filled with (round * n)'s key.
func (g *GoRedis) StuffingData(round, n int) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeOut)
	defer cancel()

	var group errgroup.Group
	for i := 0; i < round; i++ {
		// Capture the current value of i for the goroutine.
		i := i
		group.Go(func() error {
			// Expired timestamp: 1 hour ago.
			expiredTime := time.Now().Unix() - 3600
			// Future timestamp: 1 hour from now.
			futureTime := time.Now().Unix() + 3600

			for j := 0; j < n; j++ {
				// Generate key with a round number and a new UUID.
				key := fmt.Sprintf("apikey-%s-%d", uuid.NewV4().String(), i)
				var value string
				// 5 and 8 are random numbers.
				// 400 keys with a future expiration timestamp.
				// 400 keys with a dummy session and an expired timestamp.
				// 1200 keys with just an expired timestamp.
				if j%5 == 0 {
					value = fmt.Sprintf("{\"expires\": %d}", futureTime)
				} else if j%8 == 0 && j%5 != 0 {
					// Use a dummy session and an expired timestamp.
					value = fmt.Sprintf("{\"TykJWTSessionID\": \"dummy-session\", \"expires\": %d}", expiredTime)
				} else {
					value = fmt.Sprintf("{\"expires\": %d}", expiredTime)
				}

				if err := g.client.Set(ctx, key, value, 0).Err(); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return group.Wait()
}

// DBSize return DBsize of all master nodes.
func (g *GoRedis) DBSize() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeOut)
	defer cancel()
	return g.client.DBSize(ctx).Result()
}

// Password return redis password.
func (g *GoRedis) Password() string {
	return g.password
}

// Close closes the cluster client.
func (g *GoRedis) Close() error {
	return g.client.Close()
}
