package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "Single part",
			parts:    []string{"standings"},
			expected: "game-stats:standings",
		},
		{
			name:     "Multiple parts",
			parts:    []string{"division", "123", "standings"},
			expected: "game-stats:division:123:standings",
		},
		{
			name:     "Event bracket",
			parts:    []string{"event", "456", "bracket"},
			expected: "game-stats:event:456:bracket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CacheKey(tt.parts...)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCacheTTLConstants(t *testing.T) {
	assert.Equal(t, 5*time.Minute, TTLStandings)
	assert.Equal(t, 10*time.Minute, TTLBracket)
	assert.Equal(t, 2*time.Minute, TTLGameStats)
	assert.Equal(t, 5*time.Minute, TTLSpiritScores)
	assert.Equal(t, 1*time.Hour, TTLOllamaQuery)
	assert.Equal(t, 4*time.Minute, TTLSupersetToken)
	assert.Equal(t, 24*time.Hour, TTLSchemaContext)
}

// Integration tests would require a running Redis instance
// These are kept simple for unit testing
func TestRedisClient_Methods(t *testing.T) {
	// Skip if no Redis available
	t.Skip("Requires running Redis instance")

	client, err := NewRedisClient("redis://localhost:6379/0")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("Ping", func(t *testing.T) {
		err := client.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Set and Get", func(t *testing.T) {
		key := "test:key"
		value := "test value"

		err := client.Set(ctx, key, value, 1*time.Minute)
		assert.NoError(t, err)

		result, err := client.Get(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, []byte(value), result)

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("SetJSON and GetJSON", func(t *testing.T) {
		key := "test:json"
		value := map[string]interface{}{
			"name":  "Test Team",
			"score": 42,
		}

		err := client.SetJSON(ctx, key, value, 1*time.Minute)
		assert.NoError(t, err)

		var result map[string]interface{}
		err = client.GetJSON(ctx, key, &result)
		assert.NoError(t, err)
		assert.Equal(t, "Test Team", result["name"])
		assert.Equal(t, float64(42), result["score"])

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Exists", func(t *testing.T) {
		key := "test:exists"

		exists, err := client.Exists(ctx, key)
		assert.NoError(t, err)
		assert.False(t, exists)

		client.Set(ctx, key, "value", 1*time.Minute)

		exists, err = client.Exists(ctx, key)
		assert.NoError(t, err)
		assert.True(t, exists)

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("Increment", func(t *testing.T) {
		key := "test:counter"

		val, err := client.Increment(ctx, key)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), val)

		val, err = client.IncrementBy(ctx, key, 5)
		assert.NoError(t, err)
		assert.Equal(t, int64(6), val)

		// Cleanup
		client.Delete(ctx, key)
	})

	t.Run("SetNX", func(t *testing.T) {
		key := "test:lock"

		success, err := client.SetNX(ctx, key, "locked", 1*time.Minute)
		assert.NoError(t, err)
		assert.True(t, success)

		success, err = client.SetNX(ctx, key, "locked2", 1*time.Minute)
		assert.NoError(t, err)
		assert.False(t, success)

		// Cleanup
		client.Delete(ctx, key)
	})
}
