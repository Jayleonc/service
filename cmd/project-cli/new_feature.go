package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

var (
	featureNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	reservedFeatureName = map[string]struct{}{
		"feature": {},
		"server":  {},
	}
)

func runNewFeature(root string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("=== New Feature Generator ===")

	name, err := promptForFeatureName(reader)
	if err != nil {
		return err
	}

	featureType, err := promptForFeatureType(reader)
	if err != nil {
		return err
	}

	enableRBAC, err := promptForRBACOption(reader)
	if err != nil {
		return err
	}

	fmt.Printf("Creating %s feature %q...\n", featureType, name)
	if err := createFeature(root, name, featureType, enableRBAC); err != nil {
		return err
	}

	fmt.Printf("Feature %q created successfully.\n", name)
	return nil
}

func promptForFeatureName(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("Enter the new feature name (e.g. \"billing\", \"user_profile\"): ")
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}

		if input == "" {
			fmt.Println("Feature name is required.")
			continue
		}

		name := strings.ToLower(input)
		if !featureNamePattern.MatchString(name) {
			fmt.Println("Invalid feature name: only lowercase letters, digits, and underscores are allowed, and it must start with a letter.")
			continue
		}

		if _, exists := reservedFeatureName[name]; exists {
			fmt.Printf("Feature name %q is reserved. Please choose another name.\n", name)
			continue
		}

		return name, nil
	}
}

func promptForFeatureType(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("Select feature type ([1] Simple, [2] Structured): ")
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}

		choice := strings.ToLower(input)
		switch choice {
		case "", "1", "simple":
			return "simple", nil
		case "2", "structured":
			return "structured", nil
		default:
			fmt.Println("Please choose 1 for Simple or 2 for Structured.")
		}
	}
}

func promptForRBACOption(reader *bufio.Reader) (bool, error) {
	for {
		fmt.Print("Add RBAC permission declarations to routes? ([y/N]): ")
		input, err := readLine(reader)
		if err != nil {
			return false, err
		}

		choice := strings.ToLower(input)
		switch choice {
		case "", "n", "no":
			return false, nil
		case "y", "yes":
			return true, nil
		default:
			fmt.Println("Please answer yes or no (default: no).")
		}
	}
}

func createFeature(root, name, featureType string, enableRBAC bool) error {
	featureDir := filepath.Join(root, "internal", name)
	if _, err := os.Stat(featureDir); err == nil {
		return fmt.Errorf("feature directory %s already exists", featureDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check feature directory: %w", err)
	}

	if err := os.MkdirAll(featureDir, 0o755); err != nil {
		return fmt.Errorf("create feature directory: %w", err)
	}

	display := displayName(name)
	entityName := strings.ReplaceAll(display, " ", "")

	registry, err := loadErrorCodeRegistry(filepath.Join(root, "pkg", "xerr", "codes.go"))
	if err != nil {
		return fmt.Errorf("load error code registry: %w", err)
	}

	nextCodeBase := registry.nextModuleCodeBase()
	codeRangeEnd := nextCodeBase + registry.moduleRange - 1

	data := featureTemplateData{
		Package:               name,
		Name:                  name,
		FeatureName:           name,
		DisplayName:           display,
		LogMessage:            fmt.Sprintf("%s feature initialised", display),
		EntityName:            entityName,
		EntityVar:             lowerFirst(entityName),
		EnableRBAC:            enableRBAC,
		PermissionPrefix:      fmt.Sprintf("%s:", name),
		PascalName:            entityName,
		SuggestedErrorCode:    nextCodeBase,
		SuggestedErrorCodeEnd: codeRangeEnd,
	}

	files := map[string][]byte{}
	var renderErr error
	switch featureType {
	case "simple":
		files, renderErr = renderSimpleFeature(data)
	case "structured":
		files, renderErr = renderStructuredFeature(data)
	}
	if renderErr != nil {
		return renderErr
	}

	for filename, content := range files {
		path := filepath.Join(featureDir, filename)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	if err := registerModuleCodeBase(filepath.Join(root, "pkg", "xerr", "codes.go"), data.PascalName, nextCodeBase); err != nil {
		return fmt.Errorf("update error code registry: %w", err)
	}

	modulePath, err := goModulePath(root)
	if err != nil {
		return fmt.Errorf("determine module path: %w", err)
	}

	if err := updateFeaturesFile(root, modulePath, name); err != nil {
		return fmt.Errorf("update feature registry: %w", err)
	}

	return nil
}

type featureTemplateData struct {
	Package               string
	Name                  string
	FeatureName           string
	DisplayName           string
	LogMessage            string
	EntityName            string
	EntityVar             string
	EnableRBAC            bool
	PermissionPrefix      string
	PascalName            string
	SuggestedErrorCode    int
	SuggestedErrorCodeEnd int
}

type errorCodeRegistry struct {
	moduleRange int
	bases       map[string]int
}

func loadErrorCodeRegistry(path string) (*errorCodeRegistry, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse codes.go: %w", err)
	}

	registry := &errorCodeRegistry{bases: make(map[string]int)}

	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}

		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) == 0 || len(valueSpec.Values) == 0 {
				continue
			}

			lit, ok := valueSpec.Values[0].(*ast.BasicLit)
			if !ok || lit.Kind != token.INT {
				continue
			}

			v, err := strconv.Atoi(lit.Value)
			if err != nil {
				return nil, fmt.Errorf("parse constant value for %s: %w", valueSpec.Names[0].Name, err)
			}

			for _, name := range valueSpec.Names {
				switch {
				case name.Name == "ModuleCodeRange":
					registry.moduleRange = v
				case strings.HasSuffix(name.Name, "ModuleCodeBase"):
					registry.bases[name.Name] = v
				}
			}
		}
	}

	if registry.moduleRange == 0 {
		return nil, errors.New("ModuleCodeRange is not defined in codes.go")
	}

	return registry, nil
}

func (r *errorCodeRegistry) nextModuleCodeBase() int {
	highest := 0
	for _, base := range r.bases {
		if base > highest {
			highest = base
		}
	}
	if highest == 0 {
		return r.moduleRange
	}
	return highest + r.moduleRange
}

func registerModuleCodeBase(path, pascalName string, base int) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse codes.go: %w", err)
	}

	constName := fmt.Sprintf("%sModuleCodeBase", pascalName)

	var constBlock *ast.GenDecl
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}

		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}

			for _, name := range valueSpec.Names {
				if name.Name == constName {
					return fmt.Errorf("module code base %s already exists", constName)
				}
				if name.Name == "ModuleCodeRange" || strings.HasSuffix(name.Name, "ModuleCodeBase") {
					constBlock = gen
				}
			}
		}
	}

	if constBlock == nil {
		return errors.New("constants block not found in codes.go")
	}

	spec := &ast.ValueSpec{
		Names:  []*ast.Ident{ast.NewIdent(constName)},
		Values: []ast.Expr{&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(base)}},
	}

	constBlock.Specs = append(constBlock.Specs, spec)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return fmt.Errorf("format codes.go: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write codes.go: %w", err)
	}

	return nil
}

func renderSimpleFeature(data featureTemplateData) (map[string][]byte, error) {
	model, err := renderGoTemplate(modelTemplate, data)
	if err != nil {
		return nil, err
	}

	handler, err := renderGoTemplate(simpleHandlerTemplate, data)
	if err != nil {
		return nil, err
	}

	register, err := renderGoTemplate(simpleRegisterTemplate, data)
	if err != nil {
		return nil, err
	}

	errorsFile, err := renderGoTemplate(errorsTemplate, data)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		"model.go":    model,
		"handler.go":  handler,
		"register.go": register,
		"errors.go":   errorsFile,
	}, nil
}

func renderStructuredFeature(data featureTemplateData) (map[string][]byte, error) {
	model, err := renderGoTemplate(modelTemplate, data)
	if err != nil {
		return nil, err
	}

	repository, err := renderGoTemplate(structuredRepositoryTemplate, data)
	if err != nil {
		return nil, err
	}

	service, err := renderGoTemplate(structuredServiceTemplate, data)
	if err != nil {
		return nil, err
	}

	handler, err := renderGoTemplate(structuredHandlerTemplate, data)
	if err != nil {
		return nil, err
	}

	register, err := renderGoTemplate(structuredRegisterTemplate, data)
	if err != nil {
		return nil, err
	}

	errorsFile, err := renderGoTemplate(errorsTemplate, data)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		"model.go":      model,
		"repository.go": repository,
		"service.go":    service,
		"handler.go":    handler,
		"register.go":   register,
		"errors.go":     errorsFile,
	}, nil
}

func renderGoTemplate(tmpl string, data featureTemplateData) ([]byte, error) {
	t, err := template.New("feature").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("format source: %w", err)
	}

	return formatted, nil
}

func updateFeaturesFile(root, modulePath, featureName string) error {
	featuresPath := filepath.Join(root, "internal", "app", "bootstrap.go")
	content, err := os.ReadFile(featuresPath)
	if err != nil {
		return err
	}

	importPath := fmt.Sprintf("%s/internal/%s", modulePath, featureName)
	if strings.Contains(string(content), importPath) {
		return fmt.Errorf("feature %q is already registered", featureName)
	}

	updated, err := insertImport(string(content), "", importPath)
	if err != nil {
		return err
	}

	updated, err = insertFeatureEntry(updated, featureName, featureName)
	if err != nil {
		return err
	}

	formatted, err := format.Source([]byte(updated))
	if err != nil {
		return fmt.Errorf("format bootstrap.go: %w", err)
	}

	if err := os.WriteFile(featuresPath, formatted, 0o644); err != nil {
		return fmt.Errorf("write bootstrap.go: %w", err)
	}

	return nil
}

func insertImport(content, alias, path string) (string, error) {
	const importKeyword = "import ("

	idx := strings.Index(content, importKeyword)
	if idx == -1 {
		return "", errors.New("import block not found in bootstrap.go")
	}

	blockStart := idx + len(importKeyword)
	blockEnd := strings.Index(content[blockStart:], ")")
	if blockEnd == -1 {
		return "", errors.New("import block not terminated in bootstrap.go")
	}

	insertPos := blockStart + blockEnd

	var line string
	if alias != "" {
		line = fmt.Sprintf("\t%s \"%s\"\n", alias, path)
	} else {
		line = fmt.Sprintf("\t\"%s\"\n", path)
	}

	return content[:insertPos] + line + content[insertPos:], nil
}

func insertFeatureEntry(content, featureName, reference string) (string, error) {
	const marker = "var Features = []feature.Entry{"

	idx := strings.Index(content, marker)
	if idx == -1 {
		return "", errors.New("Features slice not found in bootstrap.go")
	}

	openIdx := idx + len(marker) - 1
	closingIdx, err := findClosingBrace(content, openIdx)
	if err != nil {
		return "", err
	}

	entry := fmt.Sprintf("\t{Name: \"%s\", Registrar: %s.Register},\n", featureName, reference)
	return content[:closingIdx] + entry + content[closingIdx:], nil
}

func findClosingBrace(content string, openIdx int) (int, error) {
	depth := 0
	for i := openIdx; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}

	return -1, errors.New("matching closing brace not found")
}

func displayName(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-'
	})
	if len(parts) == 0 {
		return title(name)
	}

	for i, part := range parts {
		parts[i] = title(part)
	}

	return strings.Join(parts, " ")
}

func title(word string) string {
	if word == "" {
		return ""
	}

	runes := []rune(word)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		runes[i] = unicode.ToLower(runes[i])
	}

	return string(runes)
}

func lowerFirst(word string) string {
	if word == "" {
		return ""
	}

	runes := []rune(word)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
