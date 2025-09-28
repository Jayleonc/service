package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"io/fs"
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
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	root := "."

	var err error
	switch cmd {
	case "init":
		err = runInit(root)
	case "new-module":
		err = runNewModule(root)
	case "help", "-h", "--help":
		printUsage()
		return
	default:
		fmt.Fprintf(os.Stderr, "project-cli: unknown command %q\n\n", cmd)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		exitWithError(err)
	}
}

func printUsage() {
	fmt.Println("Usage: project-cli <command>")
	fmt.Println()
	fmt.Println("Available commands:")
	fmt.Println("  init         Initialise the project with a new module path")
	fmt.Println("  new-module   Scaffold a new module interactively")
}

func runInit(root string) error {
	modulePath, err := goModulePath(root)
	if err != nil {
		return fmt.Errorf("determine current module path: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Current module path: %s\n", modulePath)

	newPath, err := promptForModulePath(reader, modulePath)
	if err != nil {
		return err
	}

	if newPath == modulePath {
		fmt.Println("Module path unchanged; nothing to do.")
		return nil
	}

	fmt.Println("Updating project files...")
	updated, err := replaceModulePath(root, modulePath, newPath)
	if err != nil {
		return err
	}

	fmt.Printf("Updated %d files.\n", updated)
	fmt.Println("Project initialisation complete. You can now comment out or remove the init-project target in the Makefile to prevent re-running it.")
	return nil
}

func promptForModulePath(reader *bufio.Reader, current string) (string, error) {
	for {
		fmt.Print("Enter the new module path: ")
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}

		if input == "" {
			fmt.Println("Module path cannot be empty. Please provide a value like github.com/company/project.")
			continue
		}

		if input == current {
			return input, nil
		}

		return input, nil
	}
}

func runNewModule(root string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("=== New Module Generator ===")

	name, err := promptForModuleName(reader)
	if err != nil {
		return err
	}

	moduleType, err := promptForModuleType(reader)
	if err != nil {
		return err
	}

	fmt.Printf("Creating %s module %q...\n", moduleType, name)
	if err := createModule(root, name, moduleType); err != nil {
		return err
	}

	fmt.Printf("Module %q created successfully.\n", name)
	return nil
}

func promptForModuleName(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("Enter the new module name (e.g. \"billing\", \"user_profile\"): ")
		input, err := readLine(reader)
		if err != nil {
			return "", err
		}

		if input == "" {
			fmt.Println("Module name is required.")
			continue
		}

		name := strings.ToLower(input)
		if !moduleNamePattern.MatchString(name) {
			fmt.Println("Invalid module name: only lowercase letters, digits, and underscores are allowed, and it must start with a letter.")
			continue
		}

		if _, exists := reservedModuleName[name]; exists {
			fmt.Printf("Module name %q is reserved. Please choose another name.\n", name)
			continue
		}

		return name, nil
	}
}

func promptForModuleType(reader *bufio.Reader) (string, error) {
	for {
		fmt.Print("Select module type ([1] Simple, [2] Structured): ")
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

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return "", err
			}
		} else {
			return "", err
		}
	}

	return strings.TrimSpace(line), nil
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "project-cli: %v\n", err)
	os.Exit(1)
}

func createModule(root, name, moduleType string) error {
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

func replaceModulePath(root, oldPath, newPath string) (int, error) {
	var updatedFiles int
	skipDirs := map[string]struct{}{
		".git":         {},
		"vendor":       {},
		"bin":          {},
		"tmp":          {},
		"node_modules": {},
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path != root {
				if _, skip := skipDirs[d.Name()]; skip {
					return filepath.SkipDir
				}
			}
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if len(data) == 0 || isBinary(data) {
			return nil
		}

		if !bytes.Contains(data, []byte(oldPath)) {
			return nil
		}

		replaced := bytes.ReplaceAll(data, []byte(oldPath), []byte(newPath))
		if bytes.Equal(data, replaced) {
			return nil
		}

		info, err := os.Stat(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(path, replaced, info.Mode()); err != nil {
			return err
		}

		updatedFiles++
		return nil
	})
	if err != nil {
		return 0, err
	}

	return updatedFiles, nil
}

func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
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

        rg.POST("{{.Route}}/ping", h.ping)
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

        rg.POST("{{.Route}}/ping", h.ping)
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
