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
	data := featureTemplateData{
		Package:          name,
		Name:             name,
		FeatureName:      name,
		DisplayName:      display,
		LogMessage:       fmt.Sprintf("%s feature initialised", display),
		EntityName:       entityName,
		EntityVar:        lowerFirst(entityName),
		EnableRBAC:       enableRBAC,
		PermissionPrefix: fmt.Sprintf("%s:", name),
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
	Package          string
	Name             string
	FeatureName      string
	DisplayName      string
	LogMessage       string
	EntityName       string
	EntityVar        string
	EnableRBAC       bool
	PermissionPrefix string
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

	return map[string][]byte{
		"model.go":    model,
		"handler.go":  handler,
		"register.go": register,
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

	return map[string][]byte{
		"model.go":      model,
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

func lowerFirst(word string) string {
	if word == "" {
		return ""
	}

	runes := []rune(word)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

const modelTemplate = `package {{.Package}}

import (
"github.com/google/uuid"

"github.com/Jayleonc/service/pkg/model"
)

// {{.EntityName}} 定义了 {{.DisplayName}} 实体的数据库结构。
type {{.EntityName}} struct {
        ID          uuid.UUID ` + "`gorm:\"type:uuid;primaryKey\"`" + `
        Name        string    ` + "`gorm:\"size:255\" json:\"name\"`" + `
        model.Base
}

// TableName overrides the default gorm table name.
func ({{.EntityName}}) TableName() string {
return "{{.Name}}"
}
`

const simpleHandlerTemplate = `package {{.Package}}

import (
        "errors"
        "io"
        "net/http"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
"gorm.io/gorm"

"github.com/Jayleonc/service/internal/feature"
"github.com/Jayleonc/service/pkg/database"
"github.com/Jayleonc/service/pkg/ginx/paginator"
"github.com/Jayleonc/service/pkg/ginx/request"
"github.com/Jayleonc/service/pkg/ginx/response"
"github.com/Jayleonc/service/pkg/xerr"
)

var (
        errInvalidPayload     = xerr.New(1, "invalid request payload")
        errRecordNotFound     = xerr.New(2, "{{.EntityVar}} not found")
        errInvalidIdentifier  = xerr.New(3, "invalid resource identifier")
        errDatabaseNotReady   = xerr.New(4, "database is not initialised")
)

// CreateInput 定义创建 {{.DisplayName}} 的请求体。
type CreateInput struct {
        Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

// GetByIDInput 定义根据 ID 查询 {{.DisplayName}} 的请求体。
type GetByIDInput struct {
        ID string ` + "`json:\"id\" binding:\"required\"`" + `
}

// UpdateInput 定义更新 {{.DisplayName}} 的请求体。
type UpdateInput struct {
        ID   string ` + "`json:\"id\" binding:\"required\"`" + `
        Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

// DeleteInput 定义删除 {{.DisplayName}} 的请求体。
type DeleteInput struct {
        ID string ` + "`json:\"id\" binding:\"required\"`" + `
}

// ListQuery 描述列表查询可用的分页和过滤参数。
type ListQuery struct {
        Page     int    ` + "`json:\"page\"`" + `
        PageSize int    ` + "`json:\"pageSize\"`" + `
        OrderBy  string ` + "`json:\"orderBy\"`" + `
        Name     string ` + "`json:\"name\"`" + `
}

type Handler struct{}

func NewHandler() *Handler {
return &Handler{}
}

// GetRoutes 在这里定义模块的相对路由，无需包含模块名。
func (h *Handler) GetRoutes() feature.ModuleRoutes {
return feature.ModuleRoutes{
AuthenticatedRoutes: []feature.RouteDefinition{
{{if .EnableRBAC}}
                {Path: "create", Handler: h.create, RequiredPermission: "{{.PermissionPrefix}}create"},
                {Path: "get_by_id", Handler: h.getByID, RequiredPermission: "{{.PermissionPrefix}}read"},
                {Path: "update", Handler: h.update, RequiredPermission: "{{.PermissionPrefix}}update"},
                {Path: "delete", Handler: h.delete, RequiredPermission: "{{.PermissionPrefix}}delete"},
                {Path: "list", Handler: h.list, RequiredPermission: "{{.PermissionPrefix}}list"},
{{else}}
                {Path: "create", Handler: h.create},
                {Path: "get_by_id", Handler: h.getByID},
                {Path: "update", Handler: h.update},
                {Path: "delete", Handler: h.delete},
                {Path: "list", Handler: h.list},
{{end}}
},
}
}

func (h *Handler) create(c *gin.Context) {
var input CreateInput
if err := c.ShouldBindJSON(&input); err != nil {
response.Error(c, http.StatusBadRequest, errInvalidPayload)
return
}

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError, errDatabaseNotReady)
return
}

        record := &{{.EntityName}}{
                ID:   uuid.New(),
                Name: input.Name,
        }

if err := db.WithContext(c.Request.Context()).Create(record).Error; err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.SuccessWithStatus(c, http.StatusCreated, record)
}

func (h *Handler) getByID(c *gin.Context) {
        var input GetByIDInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidPayload)
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidIdentifier)
                return
        }

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError, errDatabaseNotReady)
return
}

var record {{.EntityName}}
if err := db.WithContext(c.Request.Context()).First(&record, "id = ?", id).Error; err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
response.Error(c, http.StatusNotFound, errRecordNotFound)
return
}
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, record)
}

func (h *Handler) update(c *gin.Context) {
        var input UpdateInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidPayload)
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidIdentifier)
                return
        }

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError, errDatabaseNotReady)
return
}

ctx := c.Request.Context()
var record {{.EntityName}}
        if err := db.WithContext(ctx).First(&record, "id = ?", id).Error; err != nil {
                if errors.Is(err, gorm.ErrRecordNotFound) {
                        response.Error(c, http.StatusNotFound, errRecordNotFound)
                        return
                }
                response.Error(c, http.StatusInternalServerError, err)
                return
        }

        record.Name = input.Name

if err := db.WithContext(ctx).Save(&record).Error; err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, record)
}

func (h *Handler) delete(c *gin.Context) {
        var input DeleteInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidPayload)
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidIdentifier)
                return
        }

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError, errDatabaseNotReady)
return
}

if err := db.WithContext(c.Request.Context()).Delete(&{{.EntityName}}{}, "id = ?", id).Error; err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, gin.H{"id": id})
}

func (h *Handler) list(c *gin.Context) {
        var query ListQuery
        if err := c.ShouldBindJSON(&query); err != nil {
                if !errors.Is(err, io.EOF) {
                        response.Error(c, http.StatusBadRequest, errInvalidPayload)
                        return
                }
        }

db := database.Default()
if db == nil {
response.Error(c, http.StatusInternalServerError, errDatabaseNotReady)
return
}

pageReq := request.Pagination{
                Page:     query.Page,
                PageSize: query.PageSize,
                OrderBy:  query.OrderBy,
}

session := db.WithContext(c.Request.Context()).Model(&{{.EntityName}}{})
if query.Name != "" {
session = session.Where("name LIKE ?", "%"+query.Name+"%")
}

result, err := paginator.Paginate[{{.EntityName}}](session, &pageReq)
if err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, result)
}
`

const simpleRegisterTemplate = `package {{.Package}}

import (
"context"
"fmt"

"github.com/Jayleonc/service/internal/feature"
"github.com/Jayleonc/service/pkg/database"
)

func Register(ctx context.Context, deps *feature.Dependencies) error {
        // 校验当前模块所需的依赖是否已经注入。
        if err := deps.Require("DB", "Router"); err != nil {
                return fmt.Errorf("{{.Name}} feature dependencies: %w", err)
        }

        // 读取默认数据库连接，保证基础 CRUD 能正确操作数据。
        db := database.Default()
        if db == nil {
                return fmt.Errorf("{{.Name}} feature requires a configured database connection")
        }

        // 自动迁移当前模块所需的数据表结构。
        if err := db.WithContext(ctx).AutoMigrate(&{{.EntityName}}{}); err != nil {
                return fmt.Errorf("migrate {{.Name}} tables: %w", err)
        }

        // 构建处理器并注册模块路由。
        handler := NewHandler()
        // 模块的路由将统一挂载到 /v1/{{ .FeatureName }} 路径下。
        deps.Router.RegisterModule("{{ .FeatureName }}", handler.GetRoutes())

        // 记录模块初始化日志，便于排查问题。
        if deps.Logger != nil {
                deps.Logger.InfoContext(ctx, "{{.LogMessage}}", "pattern", "simple-crud")
        }

        // 返回 nil 表示初始化成功。
        return nil
}
`

const structuredRepositoryTemplate = `package {{.Package}}

import (
"context"

"github.com/google/uuid"
"gorm.io/gorm"

"github.com/Jayleonc/service/pkg/ginx/paginator"
"github.com/Jayleonc/service/pkg/ginx/request"
"github.com/Jayleonc/service/pkg/ginx/response"
)

type Repository struct {
db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
return &Repository{db: db}
}

// ListOptions 描述分页查询可用的过滤条件。
type ListOptions struct {
Pagination request.Pagination
Name       string
}

func (r *Repository) Migrate(ctx context.Context) error {
if r.db == nil {
return gorm.ErrInvalidDB
}
return r.db.WithContext(ctx).AutoMigrate(&{{.EntityName}}{})
}

func (r *Repository) Create(ctx context.Context, entity *{{.EntityName}}) error {
return r.db.WithContext(ctx).Create(entity).Error
}

func (r *Repository) Update(ctx context.Context, entity *{{.EntityName}}) error {
return r.db.WithContext(ctx).Save(entity).Error
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
return r.db.WithContext(ctx).Delete(&{{.EntityName}}{}, "id = ?", id).Error
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*{{.EntityName}}, error) {
var entity {{.EntityName}}
if err := r.db.WithContext(ctx).First(&entity, "id = ?", id).Error; err != nil {
return nil, err
}
return &entity, nil
}

func (r *Repository) List(ctx context.Context, opts ListOptions) (*response.PageResult[{{.EntityName}}], error) {
query := r.db.WithContext(ctx).Model(&{{.EntityName}}{})
if opts.Name != "" {
query = query.Where("name LIKE ?", "%"+opts.Name+"%")
}

pagination := opts.Pagination
return paginator.Paginate[{{.EntityName}}](query, &pagination)
}
`

const structuredServiceTemplate = `package {{.Package}}

import (
"context"
"fmt"

"github.com/google/uuid"

"github.com/Jayleonc/service/pkg/ginx/request"
"github.com/Jayleonc/service/pkg/ginx/response"
)

type Service struct {
repo *Repository
}

func NewService(repo *Repository) *Service {
return &Service{repo: repo}
}

// CreateParams 定义创建 {{.DisplayName}} 所需的业务参数。
type CreateParams struct {
Name string
}

// UpdateParams 定义更新 {{.DisplayName}} 所需的业务参数。
type UpdateParams struct {
ID   uuid.UUID
Name string
}

// ListParams 描述查询 {{.DisplayName}} 列表时可用的选项。
type ListParams struct {
Pagination request.Pagination
Name       string
}

func (s *Service) Create(ctx context.Context, params CreateParams) (*{{.EntityName}}, error) {
if s.repo == nil {
return nil, fmt.Errorf("repository is not configured")
}

entity := &{{.EntityName}}{
ID:   uuid.New(),
Name: params.Name,
}

if err := s.repo.Create(ctx, entity); err != nil {
return nil, err
}

return entity, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*{{.EntityName}}, error) {
if s.repo == nil {
return nil, fmt.Errorf("repository is not configured")
}

return s.repo.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, params UpdateParams) (*{{.EntityName}}, error) {
if s.repo == nil {
return nil, fmt.Errorf("repository is not configured")
}

record, err := s.repo.FindByID(ctx, params.ID)
if err != nil {
return nil, err
}

record.Name = params.Name

if err := s.repo.Update(ctx, record); err != nil {
return nil, err
}

return record, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
if s.repo == nil {
return fmt.Errorf("repository is not configured")
}

return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, params ListParams) (*response.PageResult[{{.EntityName}}], error) {
if s.repo == nil {
return nil, fmt.Errorf("repository is not configured")
}

return s.repo.List(ctx, ListOptions{
Pagination: params.Pagination,
Name:       params.Name,
})
}
`

const structuredHandlerTemplate = `package {{.Package}}

import (
        "errors"
        "io"
        "net/http"

"github.com/gin-gonic/gin"
"github.com/google/uuid"
"gorm.io/gorm"

"github.com/Jayleonc/service/internal/feature"
"github.com/Jayleonc/service/pkg/ginx/request"
"github.com/Jayleonc/service/pkg/ginx/response"
"github.com/Jayleonc/service/pkg/xerr"
)

var (
        errInvalidPayload    = xerr.New(1, "invalid request payload")
        errInvalidIdentifier = xerr.New(2, "invalid resource identifier")
        errRecordNotFound    = xerr.New(3, "{{.EntityVar}} not found")
)

type Handler struct {
service *Service
}

func NewHandler(service *Service) *Handler {
return &Handler{service: service}
}

// CreateInput 定义创建 {{.DisplayName}} 的请求结构。
type CreateInput struct {
        Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

// GetByIDInput 定义根据 ID 查询 {{.DisplayName}} 的请求结构。
type GetByIDInput struct {
        ID string ` + "`json:\"id\" binding:\"required\"`" + `
}

// UpdateInput 定义更新 {{.DisplayName}} 的请求结构。
type UpdateInput struct {
        ID   string ` + "`json:\"id\" binding:\"required\"`" + `
        Name string ` + "`json:\"name\" binding:\"required\"`" + `
}

// DeleteInput 定义删除 {{.DisplayName}} 的请求结构。
type DeleteInput struct {
        ID string ` + "`json:\"id\" binding:\"required\"`" + `
}

// ListQuery 描述查询参数。
type ListQuery struct {
        Page     int    ` + "`json:\"page\"`" + `
        PageSize int    ` + "`json:\"pageSize\"`" + `
        OrderBy  string ` + "`json:\"orderBy\"`" + `
        Name     string ` + "`json:\"name\"`" + `
}

// GetRoutes 在这里定义模块的相对路由，无需包含模块名。
func (h *Handler) GetRoutes() feature.ModuleRoutes {
return feature.ModuleRoutes{
AuthenticatedRoutes: []feature.RouteDefinition{
{{if .EnableRBAC}}
                {Path: "create", Handler: h.create, RequiredPermission: "{{.PermissionPrefix}}create"},
                {Path: "get_by_id", Handler: h.getByID, RequiredPermission: "{{.PermissionPrefix}}read"},
                {Path: "update", Handler: h.update, RequiredPermission: "{{.PermissionPrefix}}update"},
                {Path: "delete", Handler: h.delete, RequiredPermission: "{{.PermissionPrefix}}delete"},
                {Path: "list", Handler: h.list, RequiredPermission: "{{.PermissionPrefix}}list"},
{{else}}
                {Path: "create", Handler: h.create},
                {Path: "get_by_id", Handler: h.getByID},
                {Path: "update", Handler: h.update},
                {Path: "delete", Handler: h.delete},
                {Path: "list", Handler: h.list},
{{end}}
},
}
}

func (h *Handler) create(c *gin.Context) {
var input CreateInput
if err := c.ShouldBindJSON(&input); err != nil {
response.Error(c, http.StatusBadRequest, errInvalidPayload)
return
}

        entity, err := h.service.Create(c.Request.Context(), CreateParams{
                Name: input.Name,
        })
if err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.SuccessWithStatus(c, http.StatusCreated, entity)
}

func (h *Handler) getByID(c *gin.Context) {
        var input GetByIDInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidPayload)
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidIdentifier)
                return
        }

record, err := h.service.GetByID(c.Request.Context(), id)
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
response.Error(c, http.StatusNotFound, errRecordNotFound)
return
}
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, record)
}

func (h *Handler) update(c *gin.Context) {
        var input UpdateInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidPayload)
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidIdentifier)
                return
        }

        record, err := h.service.Update(c.Request.Context(), UpdateParams{
                ID:   id,
                Name: input.Name,
        })
if err != nil {
if errors.Is(err, gorm.ErrRecordNotFound) {
response.Error(c, http.StatusNotFound, errRecordNotFound)
return
}
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, record)
}

func (h *Handler) delete(c *gin.Context) {
        var input DeleteInput
        if err := c.ShouldBindJSON(&input); err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidPayload)
                return
        }

        id, err := uuid.Parse(input.ID)
        if err != nil {
                response.Error(c, http.StatusBadRequest, errInvalidIdentifier)
                return
        }

        if err := h.service.Delete(c.Request.Context(), id); err != nil {
                response.Error(c, http.StatusInternalServerError, err)
                return
        }

response.Success(c, gin.H{"id": id})
}

func (h *Handler) list(c *gin.Context) {
        var query ListQuery
        if err := c.ShouldBindJSON(&query); err != nil {
                if !errors.Is(err, io.EOF) {
                        response.Error(c, http.StatusBadRequest, errInvalidPayload)
                        return
                }
        }

result, err := h.service.List(c.Request.Context(), ListParams{
Pagination: request.Pagination{
Page:     query.Page,
PageSize: query.PageSize,
OrderBy:  query.OrderBy,
},
Name: query.Name,
})
if err != nil {
response.Error(c, http.StatusInternalServerError, err)
return
}

response.Success(c, result)
}
`

const structuredRegisterTemplate = `package {{.Package}}

import (
"context"
"fmt"

"github.com/Jayleonc/service/internal/feature"
)

func Register(ctx context.Context, deps *feature.Dependencies) error {
        // 校验当前模块所需的依赖是否已经注入。
        if err := deps.Require("DB", "Router"); err != nil {
                return fmt.Errorf("{{.Name}} feature dependencies: %w", err)
        }

        // 构建仓储层，承载数据库读写。
        repo := NewRepository(deps.DB)
        // 自动迁移当前模块所需的数据表结构。
        if err := repo.Migrate(ctx); err != nil {
                return fmt.Errorf("migrate {{.Name}} tables: %w", err)
        }

        // 初始化业务服务并装配 HTTP 处理器。
        service := NewService(repo)
        handler := NewHandler(service)

        // 模块的路由将统一挂载到 /v1/{{ .FeatureName }} 路径下。
        deps.Router.RegisterModule("{{ .FeatureName }}", handler.GetRoutes())

        // 记录模块初始化日志，便于排查问题。
        if deps.Logger != nil {
                deps.Logger.InfoContext(ctx, "{{.LogMessage}}", "pattern", "structured-crud")
        }

        // 返回 nil 表示初始化成功。
        return nil
}
`
