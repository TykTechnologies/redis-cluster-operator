package redisclustercleanup

import (
	"context"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-redis/redis/v8"
)

// processHost connects to the given Redis host and cleans up expired keys.
func processHost(host, port, password string, deleteThreshold int64, logger logr.Logger) {
	ctx := context.Background()
	addr := host + ":" + port

	// Create Redis client for this host.
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	// Prepare for SCAN iteration with a specified scan batch size.
	var cursor uint64 = 0
	scanBatchSize := int64(100) // Adjust this SCAN batch size as needed

	// keysToDelete collects keys for batch deletion.
	var keysToDelete []string

	// Regular expression to extract the "expires" value.
	expiresRegex := regexp.MustCompile(`"expires":\s*(\d+)`)

	// Begin scanning for keys matching the "apikey-*" pattern.
	for {
		keys, newCursor, err := client.Scan(ctx, cursor, "apikey-*", scanBatchSize).Result()
		if err != nil {
			logger.Error(err, "Error scanning keys", "node", addr)
			break
		}
		cursor = newCursor

		if len(keys) == 0 {
			if cursor == 0 {
				break
			}
			continue
		}

		// Use a pipeline to GET key values in a batch.
		pipe := client.Pipeline()
		cmds := make([]*redis.StringCmd, len(keys))
		for i, key := range keys {
			cmds[i] = pipe.Get(ctx, key)
		}
		_, err = pipe.Exec(ctx)
		if err != nil && !errors.Is(err, redis.Nil) {
			logger.Error(err, "Error executing pipeline GET", "node", addr)
		}

		now := time.Now().Unix()
		// Process each key's value.
		for i, key := range keys {
			value, err := cmds[i].Result()
			if err != nil {
				// Skip keys that are not found; log other errors.
				if !errors.Is(err, redis.Nil) {
					logger.Info("Error getting value for key", "key", key, "node", addr)
				}
				continue
			}
			// Skip if the value contains "TykJWTSessionID".
			if strings.Contains(value, "TykJWTSessionID") {
				continue
			}
			// Extract the expiry value using regex.
			matches := expiresRegex.FindStringSubmatch(value)
			if len(matches) < 2 {
				continue
			}
			expires, err := strconv.ParseInt(matches[1], 10, 64)
			if err != nil {
				logger.Info("Error parsing expires for key", "key", key, "node", addr)
				continue
			}
			// If the key has an expiry value > 0 and the current time is greater than expires, mark it for deletion.
			if expires > 0 && now > expires {
				keysToDelete = append(keysToDelete, key)
			}
		}

		// If the accumulated keys reach or exceed the deletion threshold, delete them using pipelining.
		logger.Info("Length of keys to delete", "redis-host", addr, "count", len(keysToDelete))
		if int64(len(keysToDelete)) >= deleteThreshold {
			logger.Info("Accumulated keys reach or exceed the deletion threshold, deleting them individually using a pipeline.", "redis-host", addr)
			delKeys := keysToDelete
			keysToDelete = nil // reset for accumulation

			pipeDel := client.Pipeline()
			for _, key := range delKeys {
				pipeDel.Del(ctx, key)
			}
			_, err := pipeDel.Exec(ctx)
			if err != nil {
				logger.Error(err, "Error executing pipeline DEL", "node", addr)
			}
		}

		// If cursor is 0 then the iteration is complete.
		if cursor == 0 {
			break
		}
	}

	// Delete any remaining keys after scanning is complete.
	if len(keysToDelete) > 0 {
		pipeDel := client.Pipeline()
		for _, key := range keysToDelete {
			pipeDel.Del(ctx, key)
		}
		_, err := pipeDel.Exec(ctx)
		if err != nil {
			logger.Error(err, "Error deleting remaining keys", "node", addr)
		}
	}
}
