
package mql2go

/*
#cgo LDFLAGS: ${SRCDIR}/mql.so
#include <stdlib.h>

extern void *tree_sitter_mql();
*/
import "C"
import (
	"sync"

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
		mqlLang = sitter.NewLanguage(ptr)
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
