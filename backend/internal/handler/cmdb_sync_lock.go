package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	cmdbSyncRedisLockKey               = "cmdb:sync:job:global:lock"
	cmdbSyncRedisLockTTLDefaultSeconds = 1800
)

var cmdbSyncRedisUnlockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`)

func (h *Handler) tryAcquireCMDBSyncLock() (release func(), acquired bool, err error) {
	release = func() {}
	if h.Redis != nil {
		lockToken := generateLockToken()
		lockTTL := h.cmdbSyncRedisLockTTL()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		locked, lockErr := h.Redis.SetNX(ctx, cmdbSyncRedisLockKey, lockToken, lockTTL).Result()
		cancel()
		if lockErr == nil {
			if !locked {
				return release, false, nil
			}
			release = func() {
				releaseCtx, releaseCancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer releaseCancel()
				if _, unlockErr := cmdbSyncRedisUnlockScript.Run(releaseCtx, h.Redis, []string{cmdbSyncRedisLockKey}, lockToken).Result(); unlockErr != nil {
					log.Printf("[cmdb] release sync redis lock failed: %v", unlockErr)
				}
			}
			return release, true, nil
		}
		log.Printf("[cmdb] acquire sync redis lock failed, fallback to DB check: %v", lockErr)
	}

	running, runningErr := h.cmdbSyncRunning()
	if runningErr != nil {
		return release, false, runningErr
	}
	if running {
		return release, false, nil
	}
	return release, true, nil
}

func (h *Handler) cmdbSyncRedisLockTTL() time.Duration {
	ttlSeconds := h.Config.CMDBSyncRedisLockTTLSeconds
	if ttlSeconds <= 0 {
		ttlSeconds = cmdbSyncRedisLockTTLDefaultSeconds
	}
	return time.Duration(ttlSeconds) * time.Second
}

func generateLockToken() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return time.Now().Format("20060102150405.000000000")
	}
	return hex.EncodeToString(buffer)
}
