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
	featureNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	reservedFeatureName = map[string]struct{}{
		"feature": {},
		"server":  {},
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
	case "new-feature":
		err = runNewFeature(root)
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
	fmt.Println("  init          Initialise the project with a new Go module path")
	fmt.Println("  new-feature   Scaffold a new feature interactively")
}

func runInit(root string) error {
	modulePath, err := goModulePath(root)
	if err != nil {
		return fmt.Errorf("determine current module path: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Current Go module path: %s\n", modulePath)

	newPath, err := promptForModulePath(reader, modulePath)
	if err != nil {
		return err
	}

	if newPath == modulePath {
		fmt.Println("Go module path unchanged; nothing to do.")
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
		fmt.Print("Enter the new Go module path: ")
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

	fmt.Printf("Creating %s feature %q...\n", featureType, name)
	if err := createFeature(root, name, featureType); err != nil {
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

func createFeature(root, name, featureType string) error {
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
	data := featureTemplateData{
		Package:     name,
		Name:        name,
		Route:       "/" + name,
		DisplayName: display,
		LogMessage:  fmt.Sprintf("%s feature initialised", display),
		EntityName:  strings.ReplaceAll(display, " ", ""),
	}

	files := map[string][]byte{}
	var err error
	switch featureType {
	case "simple":
		files, err = renderSimpleFeature(data)
	case "structured":
		files, err = renderStructuredFeature(data)
	}
	if err != nil {
		return err
	}

	for filename, content := range files {
		path := filepath.Join(featureDir, filename)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
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
	Package     string
	Name        string
	Route       string
	DisplayName string
	LogMessage  string
	EntityName  string
}

func renderSimpleFeature(data featureTemplateData) (map[string][]byte, error) {
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

func renderStructuredFeature(data featureTemplateData) (map[string][]byte, error) {
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

const simpleHandlerTemplate = `package {{.Package}}

import (
        "net/http"

        "github.com/gin-gonic/gin"

        "github.com/Jayleonc/service/internal/feature"
        "github.com/Jayleonc/service/pkg/response"
        "github.com/Jayleonc/service/pkg/xerr"
)

var (
        errEchoInvalidPayload = xerr.New(1, "invalid request payload")
)

type Handler struct{}

func NewHandler() *Handler {
        return &Handler{}
}

func (h *Handler) GetRoutes() feature.ModuleRoutes {
        return feature.ModuleRoutes{
                PublicRoutes: []feature.RouteDefinition{
                        {Path: "{{.Route}}/ping", Handler: h.ping},
                },
                AuthenticatedRoutes: []feature.RouteDefinition{
                        {Path: "{{.Route}}/echo", Handler: h.echo},
                },
        }
}

func (h *Handler) ping(c *gin.Context) {
        response.Success(c, gin.H{"message": "{{.DisplayName}} pong"})
}

func (h *Handler) echo(c *gin.Context) {
        var req struct {
                Message string ` + "`json:\"message\" binding:\"required\"`" + `
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                response.Error(c, http.StatusBadRequest, errEchoInvalidPayload)
                return
        }

        response.Success(c, gin.H{"message": req.Message})
}
`

const simpleRegisterTemplate = `package {{.Package}}

import (
        "context"
        "fmt"

        "github.com/Jayleonc/service/internal/auth"
        "github.com/Jayleonc/service/internal/feature"
)

func Register(ctx context.Context, deps feature.Dependencies) error {
        if err := deps.Require("Router"); err != nil {
                return fmt.Errorf("{{.Name}} feature dependencies: %w", err)
        }

        if auth.DefaultService() == nil {
                return fmt.Errorf("{{.Name}} feature requires the auth service to be initialised")
        }

        handler := NewHandler()
        deps.Router.RegisterModule("", handler.GetRoutes())

        if deps.Logger != nil {
                deps.Logger.InfoContext(ctx, "{{.LogMessage}}", "pattern", "singleton")
        }

        return nil
}
`

const structuredRepositoryTemplate = `package {{.Package}}

import (
        "github.com/google/uuid"
        "gorm.io/gorm"

        "github.com/Jayleonc/service/pkg/model"
)

type {{.EntityName}} struct {
        ID uuid.UUID ` + "`gorm:\"type:uuid;primaryKey\"`" + `
        model.Base
}

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

        "github.com/Jayleonc/service/internal/feature"
        "github.com/Jayleonc/service/pkg/response"
        "github.com/Jayleonc/service/pkg/xerr"
)

var (
        errServiceNotConfigured = xerr.New(1, "service not configured")
        errProcessInvalidBody   = xerr.New(2, "invalid request payload")
        errProcessFailed        = xerr.New(3, "failed to process request")
)

type Handler struct {
        service *Service
}

func NewHandler(service *Service) *Handler {
        return &Handler{service: service}
}

func (h *Handler) GetRoutes() feature.ModuleRoutes {
        return feature.ModuleRoutes{
                PublicRoutes: []feature.RouteDefinition{
                        {Path: "{{.Route}}/ping", Handler: h.ping},
                },
                AuthenticatedRoutes: []feature.RouteDefinition{
                        {Path: "{{.Route}}/process", Handler: h.process},
                },
        }
}

func (h *Handler) ping(c *gin.Context) {
        response.Success(c, gin.H{"status": "ok"})
}

func (h *Handler) process(c *gin.Context) {
        if h.service == nil {
                response.Error(c, http.StatusInternalServerError, errServiceNotConfigured)
                return
        }

        var req struct {
                Message string ` + "`json:\"message\" binding:\"required\"`" + `
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                response.Error(c, http.StatusBadRequest, errProcessInvalidBody)
                return
        }

        if err := h.service.Ping(c.Request.Context()); err != nil {
                response.Error(c, http.StatusInternalServerError, errProcessFailed)
                return
        }

        response.Success(c, gin.H{"message": req.Message})
}
`

const structuredRegisterTemplate = `package {{.Package}}

import (
        "context"
        "fmt"

        "github.com/Jayleonc/service/internal/auth"
        "github.com/Jayleonc/service/internal/feature"
)

func Register(ctx context.Context, deps feature.Dependencies) error {
        if err := deps.Require("DB", "Router"); err != nil {
                return fmt.Errorf("{{.Name}} feature dependencies: %w", err)
        }

        if auth.DefaultService() == nil {
                return fmt.Errorf("{{.Name}} feature requires the auth service to be initialised")
        }

        repo := NewRepository(deps.DB)
        svc := NewService(repo)
        handler := NewHandler(svc)
        deps.Router.RegisterModule("", handler.GetRoutes())

        if deps.Logger != nil {
                deps.Logger.InfoContext(ctx, "{{.LogMessage}}", "pattern", "structured")
        }

        return nil
}
`
