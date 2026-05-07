package golang

import (
	"strconv"
	"strings"

	"github.com/meoyawn/pggen/internal/codegen/golang/gotype"
)

// NameCompositeTranscoderFunc returns the legacy composite helper name.
func NameCompositeTranscoderFunc(typ *gotype.CompositeType) string {
	return "new" + typ.Name
}

// NameCompositeInitFunc returns the legacy composite input helper name.
func NameCompositeInitFunc(typ *gotype.CompositeType) string {
	return "new" + typ.Name + "Init"
}

// NameCompositeRawFunc returns the legacy raw composite helper name.
func NameCompositeRawFunc(typ *gotype.CompositeType) string {
	return "new" + typ.Name + "Raw"
}

// CompositeTypeDeclarer declares a new Go struct to represent a Postgres
// composite type.
type CompositeTypeDeclarer struct {
	comp *gotype.CompositeType
}

func NewCompositeTypeDeclarer(comp *gotype.CompositeType) CompositeTypeDeclarer {
	return CompositeTypeDeclarer{comp: comp}
}

func (c CompositeTypeDeclarer) DedupeKey() string {
	return "composite::" + c.comp.Name
}

func (c CompositeTypeDeclarer) Declare(pkgPath string) (string, error) {
	sb := &strings.Builder{}
	// Doc string
	if c.comp.PgComposite.Name != "" {
		sb.WriteString("// ")
		sb.WriteString(c.comp.Name)
		sb.WriteString(" represents the Postgres composite type ")
		sb.WriteString(strconv.Quote(c.comp.PgComposite.Name))
		sb.WriteString(".\n")
	}
	// Struct declaration.
	sb.WriteString("type ")
	sb.WriteString(c.comp.Name)
	sb.WriteString(" struct")
	if len(c.comp.FieldNames) == 0 {
		sb.WriteString("{") // type Foo struct{}
	} else {
		sb.WriteString(" {\n") // type Foo struct {\n
	}
	// Struct fields.
	nameLen, typeLen := getLongestNameTypes(c.comp, pkgPath)
	for i, name := range c.comp.FieldNames {
		// Name
		sb.WriteRune('\t')
		sb.WriteString(name)
		// Type
		qualType := gotype.QualifyType(c.comp.FieldTypes[i], pkgPath)
		sb.WriteString(strings.Repeat(" ", nameLen-len(name)))
		sb.WriteString(qualType)
		// JSON struct tag
		sb.WriteString(strings.Repeat(" ", typeLen-len(qualType)))
		sb.WriteString("`json:")
		sb.WriteString(strconv.Quote(c.comp.PgComposite.ColumnNames[i]))
		sb.WriteString(" db:")
		sb.WriteString(strconv.Quote(c.comp.PgComposite.ColumnNames[i]))
		sb.WriteString("`")
		sb.WriteRune('\n')
	}
	sb.WriteString("}")
	return sb.String(), nil
}

// getLongestNameTypes returns the length of the longest name and type name for
// all child fields of a composite type. Useful for aligning struct definitions.
func getLongestNameTypes(typ *gotype.CompositeType, pkgPath string) (int, int) {
	nameLen := 0
	for _, name := range typ.FieldNames {
		if n := len(name); n > nameLen {
			nameLen = n
		}
	}
	nameLen++ // 1 space to separate name from type

	typeLen := 0
	for _, childType := range typ.FieldTypes {
		if n := len(gotype.QualifyType(childType, pkgPath)); n > typeLen {
			typeLen = n
		}
	}
	typeLen++ // 1 space to separate type from struct tags.

	return nameLen, typeLen
}

// CompositeTranscoderDeclarer declares a new Go function that creates a pgx
// decoder for the Postgres type represented by the gotype.CompositeType.
type CompositeTranscoderDeclarer struct {
	typ *gotype.CompositeType
}

func NewCompositeTranscoderDeclarer(typ *gotype.CompositeType) CompositeTranscoderDeclarer {
	return CompositeTranscoderDeclarer{typ}
}

func (c CompositeTranscoderDeclarer) DedupeKey() string {
	return "type_resolver::" + c.typ.Name + "_01_transcoder"
}

func (c CompositeTranscoderDeclarer) Declare(pkgPath string) (string, error) {
	sb := &strings.Builder{}
	sb.Grow(256)
	sb.WriteString("var _ = addTypeToRegister(")
	sb.WriteString(strconv.Quote(pgTypeNameForLoadType(c.typ.PgComposite.Name)))
	sb.WriteString(")")
	return sb.String(), nil
}

// CompositeInitDeclarer is retained for the old declarer flow. pgx v5 encodes
// registered composite types directly.
type CompositeInitDeclarer struct {
	typ *gotype.CompositeType
}

func NewCompositeInitDeclarer(typ *gotype.CompositeType) CompositeInitDeclarer {
	return CompositeInitDeclarer{typ}
}

func (c CompositeInitDeclarer) DedupeKey() string {
	return "type_resolver::" + c.typ.Name + "_02_init"
}

func (c CompositeInitDeclarer) Declare(string) (string, error) {
	return "", nil
}

// CompositeRawDeclarer is retained for the old declarer flow.
type CompositeRawDeclarer struct {
	typ *gotype.CompositeType
}

func NewCompositeRawDeclarer(typ *gotype.CompositeType) CompositeRawDeclarer {
	return CompositeRawDeclarer{typ}
}

func (c CompositeRawDeclarer) DedupeKey() string {
	return "type_resolver::" + c.typ.Name + "_03_raw"
}

func (c CompositeRawDeclarer) Declare(string) (string, error) {
	return "", nil
}
