package rbac

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jayleonc/service/pkg/validation"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/pkg/ginx/response"
)

// mockService 模拟服务层响应，便于验证 Handler 行为。
type mockService struct {
	mock.Mock
}

func (m *mockService) CreateRole(ctx context.Context, input CreateRoleInput) (*Role, error) {
	args := m.Called(ctx, input)
	role, _ := args.Get(0).(*Role)
	return role, args.Error(1)
}

func (m *mockService) UpdateRole(ctx context.Context, input UpdateRoleInput) (*Role, error) {
	args := m.Called(ctx, input)
	role, _ := args.Get(0).(*Role)
	return role, args.Error(1)
}

func (m *mockService) DeleteRole(ctx context.Context, input DeleteRoleInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *mockService) ListRoles(ctx context.Context) ([]Role, error) {
	args := m.Called(ctx)
	roles, _ := args.Get(0).([]Role)
	return roles, args.Error(1)
}

func (m *mockService) AssignPermissions(ctx context.Context, input AssignRolePermissionsInput) (*Role, error) {
	args := m.Called(ctx, input)
	role, _ := args.Get(0).(*Role)
	return role, args.Error(1)
}

func (m *mockService) GetRolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	args := m.Called(ctx, roleID)
	perms, _ := args.Get(0).([]string)
	return perms, args.Error(1)
}

func (m *mockService) CreatePermission(ctx context.Context, input CreatePermissionInput) (*Permission, error) {
	args := m.Called(ctx, input)
	permission, _ := args.Get(0).(*Permission)
	return permission, args.Error(1)
}

func (m *mockService) UpdatePermission(ctx context.Context, input UpdatePermissionInput) (*Permission, error) {
	args := m.Called(ctx, input)
	permission, _ := args.Get(0).(*Permission)
	return permission, args.Error(1)
}

func (m *mockService) DeletePermission(ctx context.Context, input DeletePermissionInput) error {
	args := m.Called(ctx, input)
	return args.Error(0)
}

func (m *mockService) ListPermissions(ctx context.Context) ([]Permission, error) {
	args := m.Called(ctx)
	permissions, _ := args.Get(0).([]Permission)
	return permissions, args.Error(1)
}

func newTestRouter(svc ServiceContract) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Use a StructValidator that supports both `binding` and `validate` tags.
	binding.Validator = validation.NewDualTagValidator()
	if engine, ok := binding.Validator.Engine().(*validator.Validate); ok {
		validation.SetDefault(engine)
	} else {
		validation.Init()
	}

	handler := &Handler{svc: svc}
	router.POST("/v1/rbac/role/create", handler.createRole)
	router.POST("/v1/rbac/role/update", handler.updateRole)
	router.POST("/v1/rbac/role/delete", handler.deleteRole)
	router.POST("/v1/rbac/role/list", handler.listRoles)
	router.POST("/v1/rbac/role/assign_permissions", handler.assignRolePermissions)
	router.POST("/v1/rbac/role/get_permissions", handler.getRolePermissions)
	router.POST("/v1/rbac/permission/create", handler.createPermission)
	router.POST("/v1/rbac/permission/update", handler.updatePermission)
	router.POST("/v1/rbac/permission/delete", handler.deletePermission)
	router.POST("/v1/rbac/permission/list", handler.listPermissions)
	return router
}

func performJSONRequest(t *testing.T, router *gin.Engine, method, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		require.NoError(t, err)
	}
	req, err := http.NewRequest(method, path, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) response.Response {
	t.Helper()

	var resp response.Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	return resp
}

// TestHandlerCreateRole 覆盖角色创建接口的响应行为。
func TestHandlerCreateRole(t *testing.T) {
	cases := []struct {
		name       string
		payload    any
		prepare    func(*mockService)
		wantStatus int
	}{
		{
			name: "创建成功",
			payload: gin.H{
				"name":        "editor",
				"description": "内容编辑",
			},
			prepare: func(m *mockService) {
				m.On("CreateRole", mock.Anything, CreateRoleInput{Name: "editor", Description: "内容编辑"}).Return(&Role{Name: "EDITOR"}, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "参数缺失",
			payload:    gin.H{"description": ""},
			prepare:    func(m *mockService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "业务错误",
			payload: gin.H{"name": "editor"},
			prepare: func(m *mockService) {
				m.On("CreateRole", mock.Anything, CreateRoleInput{Name: "editor", Description: ""}).Return(nil, errors.New("failed"))
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/role/create", tc.payload)
			require.Equal(t, tc.wantStatus, recorder.Code)
			if recorder.Code == http.StatusCreated {
				resp := decodeResponse(t, recorder)
				require.Equal(t, 0, resp.Code)
			}
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerUpdateRole 验证角色更新接口的输入校验和错误映射。
func TestHandlerUpdateRole(t *testing.T) {
	roleID := uuid.New()
	name := "manager"
	desc := "运营负责人"
	cases := []struct {
		name       string
		payload    any
		prepare    func(*mockService)
		wantStatus int
	}{
		{
			name: "更新成功",
			payload: gin.H{
				"id":          roleID.String(),
				"name":        name,
				"description": desc,
			},
			prepare: func(m *mockService) {
				input := UpdateRoleInput{ID: roleID, Name: name, Description: desc}
				m.On("UpdateRole", mock.Anything, input).Return(&Role{ID: roleID, Name: "MANAGER"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "无效 UUID",
			payload:    gin.H{"id": "bad"},
			prepare:    func(m *mockService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "资源不存在",
			payload: gin.H{"id": roleID.String()},
			prepare: func(m *mockService) {
				input := UpdateRoleInput{ID: roleID}
				m.On("UpdateRole", mock.Anything, input).Return(nil, gorm.ErrRecordNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:    "业务失败",
			payload: gin.H{"id": roleID.String()},
			prepare: func(m *mockService) {
				input := UpdateRoleInput{ID: roleID}
				m.On("UpdateRole", mock.Anything, input).Return(nil, errors.New("failed"))
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/role/update", tc.payload)
			require.Equal(t, tc.wantStatus, recorder.Code)
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerDeleteRole 验证角色删除接口的分支。
func TestHandlerDeleteRole(t *testing.T) {
	roleID := uuid.New()
	cases := []struct {
		name       string
		payload    any
		prepare    func(*mockService)
		wantStatus int
	}{
		{
			name:       "删除成功",
			payload:    gin.H{"id": roleID.String()},
			prepare:    func(m *mockService) { m.On("DeleteRole", mock.Anything, DeleteRoleInput{ID: roleID}).Return(nil) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "无效参数",
			payload:    gin.H{},
			prepare:    func(m *mockService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "资源不存在",
			payload: gin.H{"id": roleID.String()},
			prepare: func(m *mockService) {
				m.On("DeleteRole", mock.Anything, DeleteRoleInput{ID: roleID}).Return(gorm.ErrRecordNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/role/delete", tc.payload)
			require.Equal(t, tc.wantStatus, recorder.Code)
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerListRoles 确认角色列表接口的返回内容。
func TestHandlerListRoles(t *testing.T) {
	cases := []struct {
		name       string
		prepare    func(*mockService)
		wantStatus int
		wantCount  int
	}{
		{
			name: "成功返回",
			prepare: func(m *mockService) {
				m.On("ListRoles", mock.Anything).Return([]Role{{Name: "ADMIN"}}, nil)
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "服务错误",
			prepare:    func(m *mockService) { m.On("ListRoles", mock.Anything).Return(nil, errors.New("failed")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/role/list", gin.H{})
			require.Equal(t, tc.wantStatus, recorder.Code)
			if recorder.Code == http.StatusOK {
				resp := decodeResponse(t, recorder)
				roles, ok := resp.Data.([]any)
				if !ok {
					raw, err := json.Marshal(resp.Data)
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(raw, &roles))
				}
				require.Len(t, roles, tc.wantCount)
			}
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerAssignRolePermissions 检查权限分配接口的返回。
func TestHandlerAssignRolePermissions(t *testing.T) {
	roleID := uuid.New()
	cases := []struct {
		name       string
		payload    any
		prepare    func(*mockService)
		wantStatus int
	}{
		{
			name: "分配成功",
			payload: gin.H{
				"roleId":      roleID.String(),
				"permissions": []string{"system:view"},
			},
			prepare: func(m *mockService) {
				input := AssignRolePermissionsInput{RoleID: roleID, Permissions: []string{"system:view"}}
				m.On("AssignPermissions", mock.Anything, input).Return(&Role{ID: roleID}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "参数缺失",
			payload:    gin.H{"roleId": roleID.String()},
			prepare:    func(m *mockService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "角色缺失",
			payload: gin.H{"roleId": roleID.String(), "permissions": []string{"system:view"}},
			prepare: func(m *mockService) {
				input := AssignRolePermissionsInput{RoleID: roleID, Permissions: []string{"system:view"}}
				m.On("AssignPermissions", mock.Anything, input).Return(nil, gorm.ErrRecordNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/role/assign_permissions", tc.payload)
			require.Equal(t, tc.wantStatus, recorder.Code)
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerGetRolePermissions 覆盖角色权限查询接口。
func TestHandlerGetRolePermissions(t *testing.T) {
	roleID := uuid.New()
	cases := []struct {
		name       string
		payload    any
		prepare    func(*mockService)
		wantStatus int
	}{
		{
			name:    "查询成功",
			payload: gin.H{"roleId": roleID.String()},
			prepare: func(m *mockService) {
				m.On("GetRolePermissions", mock.Anything, roleID).Return([]string{"system:view"}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "缺少参数",
			payload:    gin.H{},
			prepare:    func(m *mockService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "角色不存在",
			payload: gin.H{"roleId": roleID.String()},
			prepare: func(m *mockService) {
				m.On("GetRolePermissions", mock.Anything, roleID).Return(nil, gorm.ErrRecordNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/role/get_permissions", tc.payload)
			require.Equal(t, tc.wantStatus, recorder.Code)
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerCreatePermission 验证权限创建接口。
func TestHandlerCreatePermission(t *testing.T) {
	cases := []struct {
		name       string
		payload    any
		prepare    func(*mockService)
		wantStatus int
	}{
		{
			name:    "创建成功",
			payload: gin.H{"resource": "system", "action": "view"},
			prepare: func(m *mockService) {
				input := CreatePermissionInput{Resource: "system", Action: "view"}
				m.On("CreatePermission", mock.Anything, input).Return(&Permission{Resource: "system", Action: "view"}, nil)
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "缺少字段",
			payload:    gin.H{"resource": "system"},
			prepare:    func(m *mockService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "业务错误",
			payload: gin.H{"resource": "system", "action": "view"},
			prepare: func(m *mockService) {
				input := CreatePermissionInput{Resource: "system", Action: "view"}
				m.On("CreatePermission", mock.Anything, input).Return(nil, errors.New("failed"))
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/permission/create", tc.payload)
			require.Equal(t, tc.wantStatus, recorder.Code)
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerUpdatePermission 验证权限更新接口。
func TestHandlerUpdatePermission(t *testing.T) {
	permissionID := uuid.New()
	resource := "system"
	cases := []struct {
		name       string
		payload    any
		prepare    func(*mockService)
		wantStatus int
	}{
		{
			name:    "更新成功",
			payload: gin.H{"id": permissionID.String(), "resource": resource},
			prepare: func(m *mockService) {
				input := UpdatePermissionInput{ID: permissionID, Resource: resource}
				m.On("UpdatePermission", mock.Anything, input).Return(&Permission{ID: permissionID}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "无效 UUID",
			payload:    gin.H{"id": "bad"},
			prepare:    func(m *mockService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "未找到",
			payload: gin.H{"id": permissionID.String()},
			prepare: func(m *mockService) {
				input := UpdatePermissionInput{ID: permissionID}
				m.On("UpdatePermission", mock.Anything, input).Return(nil, gorm.ErrRecordNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/permission/update", tc.payload)
			require.Equal(t, tc.wantStatus, recorder.Code)
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerDeletePermission 验证权限删除接口。
func TestHandlerDeletePermission(t *testing.T) {
	permissionID := uuid.New()
	cases := []struct {
		name       string
		payload    any
		prepare    func(*mockService)
		wantStatus int
	}{
		{
			name:    "删除成功",
			payload: gin.H{"id": permissionID.String()},
			prepare: func(m *mockService) {
				m.On("DeletePermission", mock.Anything, DeletePermissionInput{ID: permissionID}).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "缺少参数",
			payload:    gin.H{},
			prepare:    func(m *mockService) {},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "未找到",
			payload: gin.H{"id": permissionID.String()},
			prepare: func(m *mockService) {
				m.On("DeletePermission", mock.Anything, DeletePermissionInput{ID: permissionID}).Return(gorm.ErrRecordNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/permission/delete", tc.payload)
			require.Equal(t, tc.wantStatus, recorder.Code)
			svc.AssertExpectations(t)
		})
	}
}

// TestHandlerListPermissions 检查权限列表接口。
func TestHandlerListPermissions(t *testing.T) {
	cases := []struct {
		name       string
		prepare    func(*mockService)
		wantStatus int
		wantCount  int
	}{
		{
			name: "返回列表",
			prepare: func(m *mockService) {
				m.On("ListPermissions", mock.Anything).Return([]Permission{{Resource: "system", Action: "view"}}, nil)
			},
			wantStatus: http.StatusOK,
			wantCount:  1,
		},
		{
			name:       "服务错误",
			prepare:    func(m *mockService) { m.On("ListPermissions", mock.Anything).Return(nil, errors.New("failed")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &mockService{}
			tc.prepare(svc)
			router := newTestRouter(svc)

			recorder := performJSONRequest(t, router, http.MethodPost, "/v1/rbac/permission/list", gin.H{})
			require.Equal(t, tc.wantStatus, recorder.Code)
			if recorder.Code == http.StatusOK {
				resp := decodeResponse(t, recorder)
				permissions, ok := resp.Data.([]any)
				if !ok {
					raw, err := json.Marshal(resp.Data)
					require.NoError(t, err)
					require.NoError(t, json.Unmarshal(raw, &permissions))
				}
				require.Len(t, permissions, tc.wantCount)
			}
			svc.AssertExpectations(t)
		})
	}
}
