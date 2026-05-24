package mdgateway

import "testing"

func TestManager_New(t *testing.T) {
	mgr := NewManager(ManagerDeps{})
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	t.Log("manager created successfully")
}
