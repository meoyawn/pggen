package golang

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/meoyawn/pggen/internal/ast"
	"github.com/meoyawn/pggen/internal/casing"
	"github.com/meoyawn/pggen/internal/codegen"
	"github.com/meoyawn/pggen/internal/codegen/golang/gotype"
	"github.com/meoyawn/pggen/internal/gomod"
)

// Templater creates query file templates.
type Templater struct {
	caser            casing.Caser
	resolver         TypeResolver
	pkg              string // Go package name
	inlineParamCount int
}

// TemplaterOpts is options to control the template logic.
type TemplaterOpts struct {
	Caser    casing.Caser
	Resolver TypeResolver
	Pkg      string // Go package name
	// How many params to inline when calling querier methods.
	InlineParamCount int
}

func NewTemplater(opts TemplaterOpts) Templater {
	return Templater{
		pkg:              opts.Pkg,
		caser:            opts.Caser,
		resolver:         opts.Resolver,
		inlineParamCount: opts.InlineParamCount,
	}
}

// TemplateAll creates query template files for each codegen.QueryFile.
func (tm Templater) TemplateAll(files []codegen.QueryFile) ([]TemplatedFile, error) {
	goQueryFiles := make([]TemplatedFile, 0, len(files))
	allDeclarers := NewDeclarerSet()

	// Pick leader file to define common structs and interfaces via Declarer.
	firstIndex := -1
	firstName := string(unicode.MaxRune)
	for i, f := range files {
		if f.SourcePath < firstName {
			firstIndex = i
			firstName = f.SourcePath
		}
	}

	for i, queryFile := range files {
		isLeader := i == firstIndex
		goFile, decls, err := tm.templateFile(queryFile, isLeader)
		if err != nil {
			return nil, fmt.Errorf("template query file %s for go: %w", queryFile.SourcePath, err)
		}
		goQueryFiles = append(goQueryFiles, goFile)
		allDeclarers.AddAll(decls.ListAll()...)
	}
	if err := assignSharedRowStructs(goQueryFiles); err != nil {
		return nil, err
	}

	// Add declarers to leader file.
	goQueryFiles[firstIndex].Declarers = allDeclarers.ListAll()

	// Remove unneeded pgconn import if possible.
	for i, file := range goQueryFiles {
		if file.needsPgconnImport() {
			continue
		}
		pgconnIdx := -1
		imports := file.Imports
		for i, pkg := range imports {
			if pkg == "github.com/jackc/pgx/v5/pgconn" {
				pgconnIdx = i
				break
			}
		}
		if pgconnIdx > -1 {
			copy(imports[pgconnIdx:], imports[pgconnIdx+1:])
			goQueryFiles[i].Imports = imports[:len(imports)-1]
		}
	}

	// Remove self imports.
	for i, file := range goQueryFiles {
		selfPkg, err := gomod.GuessPackage(file.SourcePath)
		if err != nil || selfPkg == "" {
			continue // ignore error, assume it's not a self import
		}
		selfPkgIdx := -1
		imports := file.Imports
		for i, pkg := range file.Imports {
			if pkg == selfPkg {
				selfPkgIdx = i
				break
			}
		}
		if selfPkgIdx > -1 {
			copy(imports[selfPkgIdx:], imports[selfPkgIdx+1:])
			goQueryFiles[i].Imports = imports[:len(imports)-1]
		}
	}
	return goQueryFiles, nil
}

// templateFile creates the data needed to build a Go file for a query file.
// Also returns any declarations needed by this query file. The caller must
// dedupe declarations.
func (tm Templater) templateFile(file codegen.QueryFile, isLeader bool) (TemplatedFile, DeclarerSet, error) {
	imports := NewImportSet()
	imports.AddPackage("context")
	imports.AddPackage("fmt")
	imports.AddPackage("github.com/jackc/pgx/v5/pgconn")
	if isLeader {
		imports.AddPackage("github.com/jackc/pgx/v5")
	}

	pkgPath := ""
	// NOTE: err == nil check
	// Attempt to guess package path. Ignore error if it doesn't work because
	// resolving the package isn't perfect. We'll fall back to an unqualified
	// type which will likely work since the type is probably declared in this
	// package.
	if pkg, err := gomod.GuessPackage(file.SourcePath); err == nil {
		pkgPath = pkg
	}

	queries := make([]TemplatedQuery, 0, len(file.Queries))
	declarers := NewDeclarerSet()
	for _, query := range file.Queries {
		// Build doc string.
		docs := strings.Builder{}
		avgCharsPerLine := 40
		docs.Grow(len(query.Doc) * avgCharsPerLine)
		for i, d := range query.Doc {
			if i > 0 {
				docs.WriteByte('\t') // first line is already indented in the template
			}
			docs.WriteString("// ")
			docs.WriteString(d)
			docs.WriteRune('\n')
		}

		// Build inputs.
		inputs := make([]TemplatedParam, len(query.Inputs))
		for i, input := range query.Inputs {
			goType, err := tm.resolver.Resolve(input.PgType /*nullable*/, false, pkgPath)
			if err != nil {
				return TemplatedFile{}, nil, err
			}
			imports.AddType(goType)
			inputs[i] = TemplatedParam{
				UpperName: tm.chooseUpperName(input.PgName, "UnnamedParam", i, len(query.Inputs)),
				LowerName: tm.chooseLowerName(input.PgName, "unnamedParam", i, len(query.Inputs)),
				QualType:  gotype.QualifyType(goType, pkgPath),
				Type:      goType,
				RawName:   query.Inputs[i],
			}
			ds := FindInputDeclarers(goType).ListAll()
			declarers.AddAll(ds...)
		}

		// Build outputs.
		outputs := make([]TemplatedColumn, len(query.Outputs))
		for i, out := range query.Outputs {
			goType, err := tm.resolver.Resolve(out.PgType, out.Nullable, pkgPath)
			if err != nil {
				return TemplatedFile{}, nil, err
			}
			imports.AddType(goType)
			outputs[i] = TemplatedColumn{
				PgName:    out.PgName,
				UpperName: tm.chooseUpperName(out.PgName, "UnnamedColumn", i, len(query.Outputs)),
				LowerName: tm.chooseLowerName(out.PgName, "UnnamedColumn", i, len(query.Outputs)),
				Type:      goType,
				QualType:  gotype.QualifyType(goType, pkgPath),
				Nullable:  out.Nullable,
			}
			ds := FindOutputDeclarers(goType).ListAll()
			declarers.AddAll(ds...)
		}

		nonVoidCols := removeVoidColumns(outputs)
		resultKind := query.ResultKind
		if len(nonVoidCols) == 0 {
			resultKind = ast.ResultKindExec
		}
		rowType := ""
		if query.RowType != "" {
			rowType = tm.caser.ToUpperGoIdent(query.RowType)
			if rowType == "" {
				return TemplatedFile{}, nil, fmt.Errorf("query %s has invalid row pragma %q", query.Name, query.RowType)
			}
			if resultKind == ast.ResultKindExec {
				return TemplatedFile{}, nil, fmt.Errorf("query %s uses row=%s with %s; row is only supported for result queries", query.Name, query.RowType, query.ResultKind)
			}
		}
		if resultKind != ast.ResultKindExec {
			imports.AddPackage("github.com/jackc/pgx/v5")
		}
		name := tm.caser.ToUpperGoIdent(query.Name)
		queries = append(queries, TemplatedQuery{
			Name:             name,
			RowType:          rowType,
			RowStructType:    name + "Row",
			ShouldEmitRow:    true,
			SQLVarName:       tm.caser.ToLowerGoIdent(query.Name) + "SQL",
			ResultKind:       resultKind,
			Doc:              docs.String(),
			PreparedSQL:      query.PreparedSQL,
			Inputs:           inputs,
			Outputs:          nonVoidCols,
			ScanCols:         outputs,
			InlineParamCount: tm.inlineParamCount,
		})
	}

	return TemplatedFile{
		PkgPath:    pkgPath,
		GoPkg:      tm.pkg,
		SourcePath: file.SourcePath,
		Queries:    queries,
		Imports:    imports.SortedPackages(),
		IsLeader:   isLeader,
	}, declarers, nil
}

// chooseUpperName converts pgName into a capitalized Go identifier name.
// If it's not possible to convert pgName into an identifier, uses fallback with
// a suffix using idx.
func (tm Templater) chooseUpperName(pgName string, fallback string, idx int, numOptions int) string {
	if name := tm.caser.ToUpperGoIdent(pgName); name != "" {
		return name
	}
	suffix := strconv.Itoa(idx)
	if numOptions > 9 {
		suffix = fmt.Sprintf("%2d", idx)
	}
	return fallback + suffix
}

// chooseLowerName converts pgName into an uncapitalized Go identifier name.
// If it's not possible to convert pgName into an identifier, uses fallback with
// a suffix using idx.
func (tm Templater) chooseLowerName(pgName string, fallback string, idx int, numOptions int) string {
	if name := tm.caser.ToLowerGoIdent(pgName); name != "" {
		return name
	}
	suffix := strconv.Itoa(idx)
	if numOptions > 9 {
		suffix = fmt.Sprintf("%2d", idx)
	}
	return fallback + suffix
}

// removeVoidColumns makes a copy of cols with all VoidType columns removed.
// Useful because return types shouldn't contain the void type, but we need
// to use a nil placeholder for void types when scanning a pgx.Row.
func removeVoidColumns(cols []TemplatedColumn) []TemplatedColumn {
	outs := make([]TemplatedColumn, 0, len(cols))
	for _, col := range cols {
		if _, ok := col.Type.(*gotype.VoidType); ok {
			continue
		}
		outs = append(outs, col)
	}
	return outs
}

type sharedRowShape struct {
	QueryName string
	Columns   []sharedRowColumn
}

type sharedRowColumn struct {
	PgName    string
	UpperName string
	QualType  string
	Nullable  bool
}

func assignSharedRowStructs(files []TemplatedFile) error {
	seen := make(map[string]sharedRowShape)
	for fileIdx := range files {
		for queryIdx := range files[fileIdx].Queries {
			query := &files[fileIdx].Queries[queryIdx]
			if query.RowType == "" || len(query.Outputs) <= 1 {
				continue
			}

			query.RowStructType = query.RowType + "Row"
			shape := makeSharedRowShape(*query)
			prev, ok := seen[query.RowType]
			if !ok {
				seen[query.RowType] = shape
				continue
			}
			if diff := compareSharedRowShape(prev, shape); diff != "" {
				return fmt.Errorf("row=%s used by incompatible queries %s and %s: %s", query.RowType, prev.QueryName, query.Name, diff)
			}
			query.ShouldEmitRow = false
		}
	}
	return nil
}

func makeSharedRowShape(query TemplatedQuery) sharedRowShape {
	cols := make([]sharedRowColumn, len(query.Outputs))
	for i, col := range query.Outputs {
		cols[i] = sharedRowColumn{
			PgName:    col.PgName,
			UpperName: col.UpperName,
			QualType:  col.QualType,
			Nullable:  col.Nullable,
		}
	}
	return sharedRowShape{
		QueryName: query.Name,
		Columns:   cols,
	}
}

func compareSharedRowShape(want sharedRowShape, got sharedRowShape) string {
	if len(want.Columns) != len(got.Columns) {
		return fmt.Sprintf("column count differs: %s has %d columns, %s has %d columns",
			want.QueryName, len(want.Columns), got.QueryName, len(got.Columns))
	}
	for i := range want.Columns {
		if want.Columns[i] == got.Columns[i] {
			continue
		}
		return fmt.Sprintf("column %d differs: %s has %s, %s has %s",
			i+1, want.QueryName, describeSharedRowColumn(want.Columns[i]), got.QueryName, describeSharedRowColumn(got.Columns[i]))
	}
	return ""
}

func describeSharedRowColumn(col sharedRowColumn) string {
	return fmt.Sprintf("pg=%q field=%s type=%s nullable=%t", col.PgName, col.UpperName, col.QualType, col.Nullable)
}
