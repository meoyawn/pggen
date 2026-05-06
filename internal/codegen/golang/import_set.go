package golang

import (
	"sort"

	"github.com/jschaf/pggen/internal/codegen/golang/gotype"
)

// ImportSet contains a set of imports required by one Go file.
type ImportSet struct {
	imports map[string]struct{}
}

func NewImportSet() *ImportSet {
	return &ImportSet{imports: make(map[string]struct{}, 4)}
}

// AddPackage adds a fully qualified package path to the set, like
// "github.com/jschaf/pggen/foo".
func (s *ImportSet) AddPackage(p string) {
	s.imports[p] = struct{}{}
}

// AddType adds all fully qualified package paths needed for type and any child
// types.
func (s *ImportSet) AddType(typ gotype.Type) {
	switch typ := typ.(type) {
	case *gotype.ArrayType:
		s.AddType(typ.Elem)
	case *gotype.CompositeType:
		for _, childType := range typ.FieldTypes {
			s.AddType(childType)
		}
	case *gotype.ImportType:
		s.AddPackage(typ.PkgPath)
		s.AddType(typ.Type)
	case *gotype.PointerType:
		s.AddType(typ.Elem)
	default:
		s.AddPackage(typ.Import())
	}
}

// SortedPackages returns a new slice containing the sorted packages, suitable
// for an import statement.
func (s *ImportSet) SortedPackages() []string {
	imps := make([]string, 0, len(s.imports))
	for pkg := range s.imports {
		if pkg != "" {
			imps = append(imps, pkg)
		}
	}
	sort.Strings(imps)
	return imps
}
