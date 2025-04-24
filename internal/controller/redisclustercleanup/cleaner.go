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

	"github.com/TykTechnologies/redis-cluster-operator/api/v1alpha1"
)

// processHost connects to the given Redis host and cleans up expired keys.
func processHost(host, port, password string, spec v1alpha1.RedisClusterCleanupSpec, logger logr.Logger) {
	logger.Info("processing", "host", host, "port", port)
	ctx := context.Background()
	addr := host + ":" + port

	// Create a Redis client for this host.
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	cleanupTriggered := false
	// Pre-compile expiration regexes provided in the spec.
	var expirationRegexList []*regexp.Regexp
	for _, regexStr := range spec.ExpirationRegexes {
		re, err := regexp.Compile(regexStr)
		if err != nil {
			logger.Error(err, "Invalid expiration regex", "regex", regexStr)
			continue
		}
		expirationRegexList = append(expirationRegexList, re)
	}

	scanBatchSize := int64(spec.ScanBatchSize)
	expiredThreshold := int64(spec.ExpiredThreshold)
	var keysToDelete []string

	// Iterate over each key pattern specified in the spec.
	for _, keyPattern := range spec.KeyPatterns {
		var cursor uint64 = 0
		for {
			keys, newCursor, err := client.Scan(ctx, cursor, keyPattern, scanBatchSize).Result()
			if err != nil {
				logger.Error(err, "Error scanning keys", "node", addr, "pattern", keyPattern)
				break
			}
			cursor = newCursor

			if len(keys) == 0 {
				if cursor == 0 {
					break
				}
				continue
			}

			// Get the key values via a pipeline.
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

				// Skip key if it contains any of the skip patterns.
				skipKey := false
				for _, skipPattern := range spec.SkipPatterns {
					if strings.Contains(value, skipPattern) {
						skipKey = true
						break
					}
				}
				if skipKey {
					continue
				}

				// Use the compiled expiration regexes to extract an expiration.
				var keyExpires int64
				matched := false
				for _, re := range expirationRegexList {
					matches := re.FindStringSubmatch(value)
					if len(matches) < 2 {
						continue
					}
					keyExpires, err = strconv.ParseInt(matches[1], 10, 64)
					if err != nil {
						logger.Info("Error parsing expires for key", "key", key, "node", addr)
						continue
					}
					matched = true
					break // Use the first successful match.
				}
				if !matched {
					// No valid expiration found.
					continue
				}

				// Check if the key is expired.
				if keyExpires > 0 && now > keyExpires {
					keysToDelete = append(keysToDelete, key)
				}
			}

			// Trigger deletion if the number of accumulated expired keys meets/exceeds the threshold.
			if int64(len(keysToDelete)) >= expiredThreshold {
				logger.Info("Expired keys threshold reached, deleting keys", "redis-host", addr, "pattern", keyPattern, "count", len(keysToDelete))
				cleanupTriggered = true
				pipeDel := client.Pipeline()
				for _, key := range keysToDelete {
					pipeDel.Del(ctx, key)
				}
				_, err := pipeDel.Exec(ctx)
				if err != nil {
					logger.Error(err, "Error executing pipeline DEL", "node", addr)
				}
				// Reset the key accumulator.
				keysToDelete = nil
			}

			// If SCAN iteration is complete for this pattern.
			if cursor == 0 {
				break
			}
		}
	}

	// Delete any remaining keys after scanning is complete.
	// There can be a situation where the final set of keys accumulated (from the last SCAN iteration)
	// does not meet the threshold for batch deletion. If we don't handle these,
	// they'd remain undeleted even though they meet the conditions for deletion.
	if len(keysToDelete) > 0 && cleanupTriggered {
		logger.Info("Final Cleanup: Deleting remaining keys", "redis-host", addr, "count", len(keysToDelete))
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
