package golang

import (
	"strconv"
	"strings"

	"github.com/meoyawn/pggen/internal/codegen/golang/gotype"
)

// NameArrayTranscoderFunc returns the legacy array helper name.
func NameArrayTranscoderFunc(typ *gotype.ArrayType) string {
	return "new" + typ.Elem.BaseName() + "Array"
}

// NameArrayInitFunc returns the legacy array input helper name.
func NameArrayInitFunc(typ *gotype.ArrayType) string {
	elem := typ.Elem
	if t, ok := elem.(*gotype.ImportType); ok {
		elem = t.Type
	}
	hasPtr := false
	if t, ok := elem.(*gotype.PointerType); ok {
		hasPtr = true
		elem = t.Elem
	}
	if hasPtr {
		return "new" + elem.BaseName() + "PtrArrayInit"
	} else {
		return "new" + elem.BaseName() + "ArrayInit"
	}
}

// NameArrayRawFunc returns the legacy raw array helper name.
func NameArrayRawFunc(typ *gotype.ArrayType) string {
	elem := typ.Elem
	if t, ok := elem.(*gotype.ImportType); ok {
		elem = t.Type
	}
	hasPtr := false
	if t, ok := elem.(*gotype.PointerType); ok {
		hasPtr = true
		elem = t.Elem
	}
	if hasPtr {
		return "new" + elem.BaseName() + "PtrArrayRaw"
	} else {
		return "new" + elem.BaseName() + "ArrayRaw"
	}
}

// ArrayTranscoderDeclarer declares a PostgreSQL array type registration.
type ArrayTranscoderDeclarer struct {
	typ *gotype.ArrayType
}

func NewArrayDecoderDeclarer(typ *gotype.ArrayType) ArrayTranscoderDeclarer {
	return ArrayTranscoderDeclarer{typ: typ}
}

func (a ArrayTranscoderDeclarer) DedupeKey() string {
	return "type_resolver::" + a.typ.BaseName() + "_01_transcoder"
}

func (a ArrayTranscoderDeclarer) Declare(string) (string, error) {
	sb := &strings.Builder{}
	sb.WriteString("var _ = addTypeToRegister(")
	sb.WriteString(strconv.Quote(pgTypeNameForLoadType(a.typ.PgArray.Name)))
	sb.WriteString(")")
	return sb.String(), nil
}

// ArrayInitDeclarer is retained for the old declarer flow. pgx v5 encodes
// registered array types directly.
type ArrayInitDeclarer struct {
	typ *gotype.ArrayType
}

func NewArrayInitDeclarer(typ *gotype.ArrayType) ArrayInitDeclarer {
	return ArrayInitDeclarer{typ}
}

func (a ArrayInitDeclarer) DedupeKey() string {
	return "type_resolver::" + a.typ.BaseName() + "_02_init"
}

func (a ArrayInitDeclarer) Declare(string) (string, error) {
	return "", nil
}

// ArrayRawDeclarer is retained for the old declarer flow.
type ArrayRawDeclarer struct {
	typ *gotype.ArrayType
}

func NewArrayRawDeclarer(typ *gotype.ArrayType) ArrayRawDeclarer {
	return ArrayRawDeclarer{typ}
}

func (a ArrayRawDeclarer) DedupeKey() string {
	return "type_resolver::" + a.typ.BaseName() + "_03_raw"
}

func (a ArrayRawDeclarer) Declare(pkgPath string) (string, error) {
	return "", nil
}
