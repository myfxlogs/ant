package lockutil

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// TimeoutRWMutex 带超时的读写锁
type TimeoutRWMutex struct {
	mu sync.RWMutex
}

// LockWithTimeout 尝试在指定时间内获取写锁
func (t *TimeoutRWMutex) LockWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	lockChan := make(chan struct{}, 1)
	
	go func() {
		t.mu.Lock()
		lockChan <- struct{}{}
	}()

	select {
	case <-lockChan:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to acquire lock within timeout: %v", timeout)
	}
}

// RLockWithTimeout 尝试在指定时间内获取读锁
func (t *TimeoutRWMutex) RLockWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	lockChan := make(chan struct{}, 1)
	
	go func() {
		t.mu.RLock()
		lockChan <- struct{}{}
	}()

	select {
	case <-lockChan:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("failed to acquire read lock within timeout: %v", timeout)
	}
}

// Unlock 释放写锁
func (t *TimeoutRWMutex) Unlock() {
	t.mu.Unlock()
}

// RUnlock 释放读锁
func (t *TimeoutRWMutex) RUnlock() {
	t.mu.RUnlock()
}

// LockOrderChecker 锁顺序检查器，用于防止死锁
type LockOrderChecker struct {
	orderMap map[string]int
	mu       sync.Mutex
}

// NewLockOrderChecker 创建新的锁顺序检查器
func NewLockOrderChecker() *LockOrderChecker {
	return &LockOrderChecker{
		orderMap: make(map[string]int),
	}
}

// RegisterLock 注册锁及其顺序
func (lc *LockOrderChecker) RegisterLock(lockName string, order int) {
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.orderMap[lockName] = order
}

// CheckLockOrder 检查锁获取顺序是否正确
func (lc *LockOrderChecker) CheckLockOrder(currentLock string, heldLocks []string) error {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	currentOrder, exists := lc.orderMap[currentLock]
	if !exists {
		return fmt.Errorf("lock %s not registered", currentLock)
	}

	for _, heldLock := range heldLocks {
		if heldOrder, ok := lc.orderMap[heldLock]; ok {
			if currentOrder < heldOrder {
				return fmt.Errorf("potential deadlock: trying to acquire lock %s (order %d) while holding lock %s (order %d)", 
					currentLock, currentOrder, heldLock, heldOrder)
			}
		}
	}

	return nil
}

// SafeLocker 安全锁包装器
type SafeLocker struct {
	mutex  *sync.RWMutex
	name   string
	order  int
	checker *LockOrderChecker
	heldLocks []string
}

// NewSafeLocker 创建安全锁
func NewSafeLocker(name string, order int, checker *LockOrderChecker) *SafeLocker {
	checker.RegisterLock(name, order)
	return &SafeLocker{
		mutex:   &sync.RWMutex{},
		name:    name,
		order:   order,
		checker: checker,
	}
}

// Lock 安全获取写锁
func (sl *SafeLocker) Lock() error {
	if err := sl.checker.CheckLockOrder(sl.name, sl.heldLocks); err != nil {
		return err
	}
	sl.mutex.Lock()
	sl.heldLocks = append(sl.heldLocks, sl.name)
	return nil
}

// Unlock 安全释放写锁
func (sl *SafeLocker) Unlock() {
	for i, lock := range sl.heldLocks {
		if lock == sl.name {
			sl.heldLocks = append(sl.heldLocks[:i], sl.heldLocks[i+1:]...)
			break
		}
	}
	sl.mutex.Unlock()
}

// RLock 安全获取读锁
func (sl *SafeLocker) RLock() error {
	if err := sl.checker.CheckLockOrder(sl.name, sl.heldLocks); err != nil {
		return err
	}
	sl.mutex.RLock()
	sl.heldLocks = append(sl.heldLocks, sl.name+"_read")
	return nil
}

// RUnlock 安全释放读锁
func (sl *SafeLocker) RUnlock() {
	for i, lock := range sl.heldLocks {
		if lock == sl.name+"_read" {
			sl.heldLocks = append(sl.heldLocks[:i], sl.heldLocks[i+1:]...)
			break
		}
	}
	sl.mutex.RUnlock()
}