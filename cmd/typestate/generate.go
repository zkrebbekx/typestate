// Command typestate generates compile-time-safe builders from annotated
// structs. This file holds the parse + emit core; main.go wires the CLI.
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"sort"
	"strings"
)

// fieldSpec describes one struct field the builder must handle.
type fieldSpec struct {
	Name     string // exported field name, e.g. "Name"
	TypeStr  string // rendered type, e.g. "string" or "time.Duration"
	Required bool   // tagged `typestate:"required"`
}

// builderSpec is everything needed to emit one builder.
type builderSpec struct {
	Package  string
	TypeName string
	Fields   []fieldSpec
	Imports  map[string]string // alias -> import path, for referenced field types
}

const requiredTagValue = "required"

// parse reads Go source and extracts a builderSpec for the named struct type.
func parse(filename string, src []byte, typeName string) (*builderSpec, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filename, err)
	}

	st, err := findStruct(file, typeName)
	if err != nil {
		return nil, err
	}

	// Map every import in the source so we can pull in only the packages a
	// field type actually references (keeps the generated import block tight).
	srcImports := importsByName(file)

	spec := &builderSpec{
		Package:  file.Name.Name,
		TypeName: typeName,
		Imports:  map[string]string{},
	}

	for _, f := range st.Fields.List {
		// Skip embedded fields: a builder can't address them by a simple name.
		if len(f.Names) == 0 {
			return nil, fmt.Errorf("type %s: embedded fields are not supported", typeName)
		}
		typeStr := renderType(fset, f.Type)
		required := hasRequiredTag(f.Tag)

		for _, name := range f.Names {
			if !name.IsExported() {
				// Unexported fields can't be set from another package; allow
				// them only when optional, and simply leave them out.
				if required {
					return nil, fmt.Errorf("type %s: field %q is unexported but marked required", typeName, name.Name)
				}
				continue
			}
			spec.Fields = append(spec.Fields, fieldSpec{
				Name:     name.Name,
				TypeStr:  typeStr,
				Required: required,
			})
		}

		// Record any imports the field type depends on.
		for pkg := range referencedPackages(f.Type) {
			if path, ok := srcImports[pkg]; ok {
				spec.Imports[pkg] = path
			}
		}
	}

	if len(spec.Fields) == 0 {
		return nil, fmt.Errorf("type %s: no settable fields found", typeName)
	}
	return spec, nil
}

// findStruct locates a top-level `type <name> struct{...}` declaration.
func findStruct(file *ast.File, typeName string) (*ast.StructType, error) {
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok || ts.Name.Name != typeName {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("type %s is not a struct", typeName)
			}
			return st, nil
		}
	}
	return nil, fmt.Errorf("type %s not found in source", typeName)
}

// importsByName maps the local name of each import to its path, honouring
// explicit aliases and falling back to the final path element.
func importsByName(file *ast.File) map[string]string {
	out := map[string]string{}
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		out[name] = path
	}
	return out
}

// referencedPackages returns the set of package qualifiers used in a type
// expression (e.g. "time" in time.Duration).
func referencedPackages(expr ast.Expr) map[string]struct{} {
	pkgs := map[string]struct{}{}
	ast.Inspect(expr, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok {
				pkgs[id.Name] = struct{}{}
			}
		}
		return true
	})
	return pkgs
}

func renderType(fset *token.FileSet, expr ast.Expr) string {
	var b strings.Builder
	_ = printer.Fprint(&b, fset, expr)
	return b.String()
}

func hasRequiredTag(tag *ast.BasicLit) bool {
	if tag == nil {
		return false
	}
	// tag.Value includes the surrounding back-quotes.
	st := reflectStructTag(strings.Trim(tag.Value, "`"))
	return st.get("typestate") == requiredTagValue
}

// reflectStructTag is a tiny stand-in for reflect.StructTag so we can parse
// tags from the AST without constructing a reflect value.
type reflectStructTag string

func (t reflectStructTag) get(key string) string {
	s := string(t)
	for s != "" {
		// Skip leading space.
		i := 0
		for i < len(s) && s[i] == ' ' {
			i++
		}
		s = s[i:]
		if s == "" {
			break
		}
		// Scan key up to ':'.
		i = 0
		for i < len(s) && s[i] != ':' && s[i] != ' ' && s[i] != '"' {
			i++
		}
		if i == 0 || i+1 >= len(s) || s[i] != ':' || s[i+1] != '"' {
			break
		}
		name := s[:i]
		s = s[i+2:] // skip key and `:"`
		// Scan value up to closing quote.
		i = 0
		for i < len(s) && s[i] != '"' {
			if s[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(s) {
			break
		}
		value := s[:i]
		s = s[i+1:]
		if name == key {
			return value
		}
	}
	return ""
}

// requiredNames returns the ordered list of required field names — these
// become the builder's type parameters.
func (s *builderSpec) requiredNames() []string {
	var out []string
	for _, f := range s.Fields {
		if f.Required {
			out = append(out, f.Name)
		}
	}
	return out
}

// generate renders the builder source for a spec and gofmt's it.
func generate(s *builderSpec) ([]byte, error) {
	req := s.requiredNames()

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "// Code generated by \"typestate -type %s\"; DO NOT EDIT.\n\n", s.TypeName)
	fmt.Fprintf(&buf, "package %s\n\n", s.Package)

	// Imports: always the runtime, plus any field-type packages, sorted.
	fmt.Fprintf(&buf, "import (\n\tts \"github.com/zkrebbekx/typestate\"\n")
	var extra []string
	for _, path := range s.Imports {
		extra = append(extra, path)
	}
	sort.Strings(extra)
	for _, path := range extra {
		fmt.Fprintf(&buf, "\t%q\n", path)
	}
	buf.WriteString(")\n\n")

	builder := s.TypeName + "Builder"
	paramDecl := ""  // "[Name, URL ts.Flag]"
	paramNames := "" // "[Name, URL]"
	allSet := ""     // "[ts.Set, ts.Set]"
	allUnset := ""   // "[ts.Unset, ts.Unset]"
	if len(req) > 0 {
		paramDecl = "[" + strings.Join(req, ", ") + " ts.Flag]"
		paramNames = "[" + strings.Join(req, ", ") + "]"
		set := make([]string, len(req))
		unset := make([]string, len(req))
		for i := range req {
			set[i] = "ts.Set"
			unset[i] = "ts.Unset"
		}
		allSet = "[" + strings.Join(set, ", ") + "]"
		allUnset = "[" + strings.Join(unset, ", ") + "]"
	}

	// Builder type.
	fmt.Fprintf(&buf, "// %s builds a %s, tracking required fields in its type parameters.\n", builder, s.TypeName)
	fmt.Fprintf(&buf, "type %s%s struct{ v %s }\n\n", builder, paramDecl, s.TypeName)

	// Constructor.
	fmt.Fprintf(&buf, "// New%s returns a builder with every required field unset.\n", builder)
	fmt.Fprintf(&buf, "func New%s() %s%s {\n\treturn %s%s{}\n}\n\n", builder, builder, allUnset, builder, allUnset)

	// WithX methods.
	for _, f := range s.Fields {
		method := "With" + f.Name
		if f.Required {
			ret := returnInstantiation(req, f.Name)
			fmt.Fprintf(&buf, "// %s sets the required field %s.\n", method, f.Name)
			fmt.Fprintf(&buf, "func (b %s%s) %s(v %s) %s%s {\n", builder, paramNames, method, f.TypeStr, builder, ret)
			fmt.Fprintf(&buf, "\tb.v.%s = v\n\treturn %s%s(b)\n}\n\n", f.Name, builder, ret)
		} else {
			fmt.Fprintf(&buf, "// %s sets the optional field %s.\n", method, f.Name)
			fmt.Fprintf(&buf, "func (b %s%s) %s(v %s) %s%s {\n", builder, paramNames, method, f.TypeStr, builder, paramNames)
			fmt.Fprintf(&buf, "\tb.v.%s = v\n\treturn b\n}\n\n", f.Name)
		}
	}

	// Terminal guard.
	fmt.Fprintf(&buf, "// Build%s returns the built %s. It only compiles once every\n// required field has been set.\n", s.TypeName, s.TypeName)
	fmt.Fprintf(&buf, "func Build%s(b %s%s) %s {\n\treturn b.v\n}\n", s.TypeName, builder, allSet, s.TypeName)

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gofmt generated source for %s: %w\n%s", s.TypeName, err, buf.String())
	}
	return formatted, nil
}

// returnInstantiation builds the type-argument list for a required field's
// WithX method: every slot passes through except the one being set, which
// becomes ts.Set.
func returnInstantiation(req []string, setField string) string {
	args := make([]string, len(req))
	for i, name := range req {
		if name == setField {
			args[i] = "ts.Set"
		} else {
			args[i] = name
		}
	}
	return "[" + strings.Join(args, ", ") + "]"
}
