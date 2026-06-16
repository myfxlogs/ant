package asset_analysis

import (
	"testing"

	"go.uber.org/zap"
)

func TestNewAssetAnalysisServer_Init(t *testing.T) {
	t.Parallel()
	srv := NewAssetAnalysisServer(nil, nil, nil, zap.NewNop())
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
	if srv.analyzer != nil {
		t.Fatal("expected nil analyzer when nil passed")
	}
	if srv.aiSvc != nil {
		t.Fatal("expected nil aiSvc when nil passed")
	}
	if srv.platformSvc != nil {
		t.Fatal("expected nil platformSvc when nil passed")
	}
	if srv.log == nil {
		t.Fatal("expected non-nil logger")
	}
}
