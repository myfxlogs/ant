package coordination

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Lease struct {
	AccountID string
	Owner     string
	Fence     int64

	ctx    context.Context
	cancel context.CancelFunc
}

func (l *Lease) Done() <-chan struct{} {
	if l == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	return l.ctx.Done()
}

func (l *Lease) Cancel() {
	if l == nil {
		return
	}
	l.cancel()
}

type SessionLeader struct {
	rdb *redis.Client

	instanceID string
	prefix     string

	leaseTTL      time.Duration
	renewInterval time.Duration
}

func NewSessionLeader(rdb *redis.Client, instanceID string) *SessionLeader {
	return &SessionLeader{
		rdb:           rdb,
		instanceID:    instanceID,
		prefix:        "antrader:session_leader:",
		leaseTTL:      20 * time.Second,
		renewInterval: 5 * time.Second,
	}
}

func (s *SessionLeader) SetPrefix(prefix string) {
	if s == nil {
		return
	}
	if prefix != "" {
		s.prefix = prefix
	}
}

func (s *SessionLeader) SetLease(ttl time.Duration, renewInterval time.Duration) {
	if s == nil {
		return
	}
	if ttl > 0 {
		s.leaseTTL = ttl
	}
	if renewInterval > 0 {
		s.renewInterval = renewInterval
	}
}

func (s *SessionLeader) leaderKey(accountID string) string {
	return s.prefix + "account:" + accountID + ":leader"
}

func (s *SessionLeader) fenceKey(accountID string) string {
	return s.prefix + "account:" + accountID + ":fence"
}

var acquireScript = redis.NewScript(`
local leaderKey = KEYS[1]
local fenceKey = KEYS[2]
local instanceID = ARGV[1]
local ttlMs = tonumber(ARGV[2])

if redis.call('EXISTS', leaderKey) == 1 then
  return {0, redis.call('GET', leaderKey)}
end

local fence = redis.call('INCR', fenceKey)
local owner = instanceID .. '|' .. tostring(fence)
redis.call('SET', leaderKey, owner, 'PX', ttlMs)
return {fence, owner}
`)

var renewScript = redis.NewScript(`
local leaderKey = KEYS[1]
local owner = ARGV[1]
local ttlMs = tonumber(ARGV[2])

if redis.call('GET', leaderKey) == owner then
  redis.call('PEXPIRE', leaderKey, ttlMs)
  return 1
end
return 0
`)

var releaseScript = redis.NewScript(`
local leaderKey = KEYS[1]
local owner = ARGV[1]

if redis.call('GET', leaderKey) == owner then
  redis.call('DEL', leaderKey)
  return 1
end
return 0
`)

func (s *SessionLeader) TryAcquire(ctx context.Context, accountID string) (*Lease, bool, error) {
	if s == nil || s.rdb == nil {
		return nil, false, fmt.Errorf("redis leader not configured")
	}
	if accountID == "" {
		return nil, false, fmt.Errorf("accountID required")
	}

	res, err := acquireScript.Run(ctx, s.rdb, []string{s.leaderKey(accountID), s.fenceKey(accountID)}, s.instanceID, s.leaseTTL.Milliseconds()).Result()
	if err != nil {
		return nil, false, err
	}

	arr, ok := res.([]interface{})
	if !ok || len(arr) < 2 {
		return nil, false, fmt.Errorf("unexpected acquire result: %T", res)
	}

	fence, _ := arr[0].(int64)
	owner, _ := arr[1].(string)
	if fence == 0 {
		return nil, false, nil
	}

	lctx, cancel := context.WithCancel(context.Background())
	lease := &Lease{AccountID: accountID, Owner: owner, Fence: fence, ctx: lctx, cancel: cancel}
	go s.renewLoop(lease)
	return lease, true, nil
}

func (s *SessionLeader) renewLoop(lease *Lease) {
	if s == nil || lease == nil {
		return
	}
	t := time.NewTicker(s.renewInterval)
	defer t.Stop()

	for {
		select {
		case <-lease.ctx.Done():
			return
		case <-t.C:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			ok, err := renewScript.Run(ctx, s.rdb, []string{s.leaderKey(lease.AccountID)}, lease.Owner, s.leaseTTL.Milliseconds()).Int()
			cancel()
			if err != nil || ok != 1 {
				lease.cancel()
				return
			}
		}
	}
}

func (s *SessionLeader) Release(ctx context.Context, lease *Lease) {
	if s == nil || s.rdb == nil || lease == nil {
		return
	}
	lease.cancel()
	_, _ = releaseScript.Run(ctx, s.rdb, []string{s.leaderKey(lease.AccountID)}, lease.Owner).Result()
}
