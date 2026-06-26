package mql2go

/*
#cgo LDFLAGS: -L/opt/ant/tools/mql_transpiler/grammar/mql -l:mql.so -Wl,-rpath,/opt/ant/tools/mql_transpiler/grammar/mql
#include <stdlib.h>

// tree_sitter_mql is exported by mql.so
extern void *tree_sitter_mql();
*/
import "C"
import (
	"sync"
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

var (
	langOnce sync.Once
	mqlLang  *sitter.Language
)

// Language returns the MQL tree-sitter language.
// The grammar is loaded once from the shared object.
func Language() (*sitter.Language, error) {
	var err error
	langOnce.Do(func() {
		ptr := C.tree_sitter_mql()
		if ptr == nil {
			err = wrapErr("tree_sitter_mql returned nil")
			return
		}
		mqlLang = sitter.NewLanguage(unsafe.Pointer(ptr))
	})
	if err != nil {
		return nil, err
	}
	if mqlLang == nil {
		return nil, wrapErr("could not create MQL language")
	}
	return mqlLang, nil
}

func wrapErr(msg string) error { return &mqlErr{msg} }

type mqlErr struct{ msg string }

func (e *mqlErr) Error() string { return "mql2go: " + e.msg }
