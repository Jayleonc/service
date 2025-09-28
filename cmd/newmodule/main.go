package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

var (
	moduleNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	reservedModuleName = map[string]struct{}{
		"module": {},
		"server": {},
	}
)

func main() {
	var (
		nameArg = flag.String("name", "", "module name (lowercase, underscores allowed)")
		typeArg = flag.String("type", "", "module type: simple or structured")
		rootArg = flag.String("root", ".", "project root directory")
	)

	flag.Parse()

	name := strings.TrimSpace(*nameArg)
	moduleType := strings.ToLower(strings.TrimSpace(*typeArg))
	root := strings.TrimSpace(*rootArg)

	if name == "" {
		exitWithError(errors.New("module name is required"))
	}

	name = strings.ToLower(name)
	if !moduleNamePattern.MatchString(name) {
		exitWithError(fmt.Errorf("module name %q is invalid: only lowercase letters, digits and underscores are allowed", name))
	}

	if moduleType != "simple" && moduleType != "structured" {
		exitWithError(fmt.Errorf("module type %q is invalid: must be simple or structured", moduleType))
	}

	if root == "" {
		root = "."
	}

	if _, exists := reservedModuleName[name]; exists {
		exitWithError(fmt.Errorf("module name %q is reserved", name))
	}

	if err := run(root, name, moduleType); err != nil {
		exitWithError(err)
	}
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "newmodule: %v\n", err)
	os.Exit(1)
}

func run(root, name, moduleType string) error {
	moduleDir := filepath.Join(root, "internal", name)
	if _, err := os.Stat(moduleDir); err == nil {
		return fmt.Errorf("module directory %s already exists", moduleDir)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("check module directory: %w", err)
	}

	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		return fmt.Errorf("create module directory: %w", err)
	}

	data := templateData{
		Package:     name,
		Name:        name,
		Route:       "/" + name,
		DisplayName: displayName(name),
		LogMessage:  fmt.Sprintf("%s module initialised", displayName(name)),
	}

	files := map[string][]byte{}
	var err error
	switch moduleType {
	case "simple":
		files, err = renderSimpleModule(data)
	case "structured":
		files, err = renderStructuredModule(data)
	}
	if err != nil {
		return err
	}

	for filename, content := range files {
		path := filepath.Join(moduleDir, filename)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
	}

	modulePath, err := goModulePath(root)
	if err != nil {
		return fmt.Errorf("determine module path: %w", err)
	}

	if err := updateModulesFile(root, modulePath, name); err != nil {
		return fmt.Errorf("update module registry: %w", err)
	}

	return nil
}

type templateData struct {
	Package     string
	Name        string
	Route       string
	DisplayName string
	LogMessage  string
}

func renderSimpleModule(data templateData) (map[string][]byte, error) {
	handler, err := renderGoTemplate(simpleHandlerTemplate, data)
	if err != nil {
		return nil, err
	}

	register, err := renderGoTemplate(simpleRegisterTemplate, data)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		"handler.go":  handler,
		"register.go": register,
	}, nil
}

func renderStructuredModule(data templateData) (map[string][]byte, error) {
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

	return map[string][]byte{
		"repository.go": repository,
		"service.go":    service,
		"handler.go":    handler,
		"register.go":   register,
	}, nil
}

func renderGoTemplate(tmpl string, data templateData) ([]byte, error) {
	t, err := template.New("module").Parse(tmpl)
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

func goModulePath(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}

	return "", errors.New("module path not found in go.mod")
}

func updateModulesFile(root, modulePath, moduleName string) error {
	modulesPath := filepath.Join(root, "internal", "server", "modules.go")
	content, err := os.ReadFile(modulesPath)
	if err != nil {
		return err
	}

	importPath := fmt.Sprintf("%s/internal/%s", modulePath, moduleName)
	if strings.Contains(string(content), importPath) {
		return fmt.Errorf("module %q is already registered", moduleName)
	}

	alias := fmt.Sprintf("%smodule", moduleName)
	reference := alias
	if reference == "" {
		reference = moduleName
	}

	updated, err := insertImport(string(content), alias, importPath)
	if err != nil {
		return err
	}

	updated, err = insertModuleEntry(updated, moduleName, reference)
	if err != nil {
		return err
	}

	formatted, err := format.Source([]byte(updated))
	if err != nil {
		return fmt.Errorf("format modules.go: %w", err)
	}

	if err := os.WriteFile(modulesPath, formatted, 0o644); err != nil {
		return fmt.Errorf("write modules.go: %w", err)
	}

	return nil
}

func insertImport(content, alias, path string) (string, error) {
	const importKeyword = "import ("

	idx := strings.Index(content, importKeyword)
	if idx == -1 {
		return "", errors.New("import block not found in modules.go")
	}

	blockStart := idx + len(importKeyword)
	blockEnd := strings.Index(content[blockStart:], ")")
	if blockEnd == -1 {
		return "", errors.New("import block not terminated in modules.go")
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

func insertModuleEntry(content, moduleName, reference string) (string, error) {
	const marker = "var Modules = []module.Entry{"

	idx := strings.Index(content, marker)
	if idx == -1 {
		return "", errors.New("Modules slice not found in modules.go")
	}

	openIdx := idx + len(marker) - 1
	closingIdx, err := findClosingBrace(content, openIdx)
	if err != nil {
		return "", err
	}

	entry := fmt.Sprintf("\t{Name: \"%s\", Registrar: %s.Register},\n", moduleName, reference)
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

const simpleHandlerTemplate = `package {{.Package}}

import (
        "net/http"

        "github.com/gin-gonic/gin"

        "github.com/Jayleonc/service/pkg/database"
        "github.com/Jayleonc/service/pkg/logger"
)

type Handler struct{}

func NewHandler() *Handler {
        return &Handler{}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
        if rg == nil {
                return
        }

        rg.GET("{{.Route}}/ping", h.ping)
}

func (h *Handler) ping(c *gin.Context) {
        log := logger.Default()
        if log != nil {
                log.Debug("{{.DisplayName}} ping received")
        }

        db := database.Default()
        if db == nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "database not initialised"})
                return
        }

        c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
`

const simpleRegisterTemplate = `package {{.Package}}

import (
        "context"
        "fmt"

        "github.com/Jayleonc/service/internal/module"
)

func Register(ctx context.Context, deps module.Dependencies) error {
        if deps.API == nil {
                return fmt.Errorf("{{.Name}} module requires the API router group")
        }

        handler := NewHandler()
        handler.RegisterRoutes(deps.API)

        if deps.Logger != nil {
                deps.Logger.InfoContext(ctx, "{{.LogMessage}}", "pattern", "singleton")
        }

        return nil
}
`

const structuredRepositoryTemplate = `package {{.Package}}

import "gorm.io/gorm"

type Repository struct {
        db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
        return &Repository{db: db}
}
`

const structuredServiceTemplate = `package {{.Package}}

import (
        "context"
        "fmt"
)

type Service struct {
        repo *Repository
}

func NewService(repo *Repository) *Service {
        return &Service{repo: repo}
}

func (s *Service) Ping(ctx context.Context) error {
        if ctx == nil {
                return fmt.Errorf("context is required")
        }
        if s.repo == nil {
                return fmt.Errorf("repository not configured")
        }
        return nil
}
`

const structuredHandlerTemplate = `package {{.Package}}

import (
        "net/http"

        "github.com/gin-gonic/gin"
)

type Handler struct {
        service *Service
}

func NewHandler(service *Service) *Handler {
        return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
        if rg == nil {
                return
        }

        rg.GET("{{.Route}}/ping", h.ping)
}

func (h *Handler) ping(c *gin.Context) {
        if h.service == nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": "service not configured"})
                return
        }

        if err := h.service.Ping(c.Request.Context()); err != nil {
                c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
                return
        }

        c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
`

const structuredRegisterTemplate = `package {{.Package}}

import (
        "context"
        "fmt"

        "github.com/Jayleonc/service/internal/module"
)

func Register(ctx context.Context, deps module.Dependencies) error {
        if deps.DB == nil {
                return fmt.Errorf("{{.Name}} module requires a database instance")
        }
        if deps.API == nil {
                return fmt.Errorf("{{.Name}} module requires the API router group")
        }

        repo := NewRepository(deps.DB)
        svc := NewService(repo)
        handler := NewHandler(svc)
        handler.RegisterRoutes(deps.API)

        if deps.Logger != nil {
                deps.Logger.InfoContext(ctx, "{{.LogMessage}}", "pattern", "structured")
        }

        return nil
}
`
