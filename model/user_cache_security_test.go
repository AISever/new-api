package model

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/require"
)

func TestUserAccessTokenNeverSerializes(t *testing.T) {
	secret := "security-test-access-token"
	user := User{Id: 7, AccessToken: &secret}
	data, err := common.Marshal(user)
	require.NoError(t, err)
	require.NotContains(t, string(data), secret)
	require.NotContains(t, string(data), "access_token")
	require.Equal(t, secret, user.GetAccessToken(), "explicit token retrieval stays available")
}

// This integration test uses a dedicated Redis instance, never a production DB.
func TestRedisUserSnapshotCannotRestoreConsumedQuota(t *testing.T) {
	addr := os.Getenv("SECURITY_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set SECURITY_TEST_REDIS_ADDR to an isolated Redis")
	}
	setupInviteUserTestDB(t)
	oldClient, oldEnabled := common.RDB, common.RedisEnabled
	oldFrequency := common.SyncFrequency
	common.SyncFrequency = 60
	client := redis.NewClient(&redis.Options{Addr: addr})
	common.RDB, common.RedisEnabled = client, true
	ctx := context.Background()
	user := User{Id: 700131, Username: "security-cache-user", Password: "test", Group: "default", Status: common.UserStatusEnabled, Quota: 1000}
	key := getUserCacheKey(user.Id)
	t.Cleanup(func() {
		client.Del(ctx, key)
		client.Close()
		common.RDB, common.RedisEnabled = oldClient, oldEnabled
		common.SyncFrequency = oldFrequency
	})
	require.NoError(t, client.Ping(ctx).Err())
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, populateUserCache(user))
	var wg sync.WaitGroup
	errors := make(chan error, 300)
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() { defer wg.Done(); errors <- client.HIncrBy(ctx, key, "Quota", -1).Err() }()
		go func() { defer wg.Done(); errors <- updateUserCache(user) }()
		go func() { defer wg.Done(); errors <- populateUserCache(user) }()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	quota, err := client.HGet(ctx, key, "Quota").Int()
	require.NoError(t, err)
	require.Equal(t, 900, quota)
	require.NoError(t, UpdateUserSetting(user.Id, dto.UserSetting{Language: "en"}))
	quota, err = client.HGet(ctx, key, "Quota").Int()
	require.NoError(t, err)
	require.Equal(t, 900, quota)
	require.NoError(t, updateUserStatusCache(user.Id, false))
	require.NoError(t, populateUserCache(user))
	status, err := client.HGet(ctx, key, "Status").Int()
	require.NoError(t, err)
	require.Equal(t, common.UserStatusDisabled, status)
}
