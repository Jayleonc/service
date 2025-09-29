package rbac

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Jayleonc/service/pkg/constant"
)

// mockRepository 使用 testify 模拟仓储层行为。
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) Migrate(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockRepository) CreateRole(ctx context.Context, role *Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *mockRepository) UpdateRole(ctx context.Context, role *Role) error {
	args := m.Called(ctx, role)
	return args.Error(0)
}

func (m *mockRepository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockRepository) ListRoles(ctx context.Context) ([]Role, error) {
	args := m.Called(ctx)
	roles, _ := args.Get(0).([]Role)
	return roles, args.Error(1)
}

func (m *mockRepository) FindRoleByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	args := m.Called(ctx, id)
	role, _ := args.Get(0).(*Role)
	return role, args.Error(1)
}

func (m *mockRepository) FindRoleByName(ctx context.Context, name string) (*Role, error) {
	args := m.Called(ctx, name)
	role, _ := args.Get(0).(*Role)
	return role, args.Error(1)
}

func (m *mockRepository) FindRolesByNames(ctx context.Context, names []string) ([]*Role, error) {
	args := m.Called(ctx, names)
	roles, _ := args.Get(0).([]*Role)
	return roles, args.Error(1)
}

func (m *mockRepository) CreatePermission(ctx context.Context, permission *Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *mockRepository) UpdatePermission(ctx context.Context, permission *Permission) error {
	args := m.Called(ctx, permission)
	return args.Error(0)
}

func (m *mockRepository) DeletePermission(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockRepository) ListPermissions(ctx context.Context) ([]Permission, error) {
	args := m.Called(ctx)
	perms, _ := args.Get(0).([]Permission)
	return perms, args.Error(1)
}

func (m *mockRepository) FindPermissionByID(ctx context.Context, id uuid.UUID) (*Permission, error) {
	args := m.Called(ctx, id)
	permission, _ := args.Get(0).(*Permission)
	return permission, args.Error(1)
}

func (m *mockRepository) FindPermissionsByKeys(ctx context.Context, keys []string) ([]*Permission, error) {
	args := m.Called(ctx, keys)
	permissions, _ := args.Get(0).([]*Permission)
	return permissions, args.Error(1)
}

func (m *mockRepository) ReplaceRolePermissions(ctx context.Context, role *Role, permissions []*Permission) error {
	args := m.Called(ctx, role, permissions)
	return args.Error(0)
}

func (m *mockRepository) UserHasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	args := m.Called(ctx, userID, permission)
	allowed, _ := args.Get(0).(bool)
	return allowed, args.Error(1)
}

func newMockService(repo *mockRepository) *Service {
	return &Service{repo: repo}
}

// TestServiceCreateRole 使用表驱动测试角色创建流程。
func TestServiceCreateRole(t *testing.T) {
	cases := []struct {
		name    string
		input   CreateRoleInput
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name:  "成功创建并规范化名称",
			input: CreateRoleInput{Name: "  manager ", Description: "业务管理员"},
			prepare: func(m *mockRepository) {
				m.On("CreateRole", mock.Anything, mock.MatchedBy(func(role *Role) bool {
					return role.Name == "MANAGER" && role.Description == "业务管理员"
				})).Return(nil)
			},
		},
		{
			name:    "缺失名称触发校验错误",
			input:   CreateRoleInput{Name: "   "},
			prepare: func(m *mockRepository) {},
			wantErr: true,
		},
		{
			name:  "仓储返回错误",
			input: CreateRoleInput{Name: "auditor"},
			prepare: func(m *mockRepository) {
				m.On("CreateRole", mock.Anything, mock.AnythingOfType("*rbac.Role")).Return(errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			if tc.prepare != nil {
				tc.prepare(mockRepo)
			}
			svc := newMockService(mockRepo)

			role, err := svc.CreateRole(context.Background(), tc.input)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, role)
			} else {
				require.NoError(t, err)
				require.NotNil(t, role)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceUpdateRole 覆盖角色更新的正常与异常路径。
func TestServiceUpdateRole(t *testing.T) {
	newName := " leader "
	newDescription := "部门负责人"
	cases := []struct {
		name    string
		input   UpdateRoleInput
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name: "成功更新字段",
			input: UpdateRoleInput{
				ID:          uuid.New(),
				Name:        &newName,
				Description: &newDescription,
			},
			prepare: func(m *mockRepository) {
				existing := &Role{ID: uuid.New(), Name: "OLD", Description: "描述"}
				m.On("FindRoleByID", mock.Anything, mock.Anything).Return(existing, nil)
				m.On("UpdateRole", mock.Anything, mock.MatchedBy(func(role *Role) bool {
					return role.Name == "LEADER" && role.Description == "部门负责人"
				})).Return(nil)
			},
		},
		{
			name:  "空名称触发错误",
			input: UpdateRoleInput{ID: uuid.New(), Name: func() *string { v := " "; return &v }()},
			prepare: func(m *mockRepository) {
				existing := &Role{ID: uuid.New(), Name: "OLD"}
				m.On("FindRoleByID", mock.Anything, mock.Anything).Return(existing, nil)
			},
			wantErr: true,
		},
		{
			name:  "查询失败",
			input: UpdateRoleInput{ID: uuid.New()},
			prepare: func(m *mockRepository) {
				m.On("FindRoleByID", mock.Anything, mock.Anything).Return(nil, errors.New("db"))
			},
			wantErr: true,
		},
		{
			name:  "保存失败",
			input: UpdateRoleInput{ID: uuid.New()},
			prepare: func(m *mockRepository) {
				existing := &Role{ID: uuid.New(), Name: "OLD"}
				m.On("FindRoleByID", mock.Anything, mock.Anything).Return(existing, nil)
				m.On("UpdateRole", mock.Anything, mock.AnythingOfType("*rbac.Role")).Return(errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			role, err := svc.UpdateRole(context.Background(), tc.input)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, role)
			} else {
				require.NoError(t, err)
				require.NotNil(t, role)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceDeleteRole 验证删除逻辑能正确传递参数并处理错误。
func TestServiceDeleteRole(t *testing.T) {
	roleID := uuid.New()
	cases := []struct {
		name    string
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name: "删除成功",
			prepare: func(m *mockRepository) {
				m.On("DeleteRole", mock.Anything, roleID).Return(nil)
			},
		},
		{
			name: "删除失败返回错误",
			prepare: func(m *mockRepository) {
				m.On("DeleteRole", mock.Anything, roleID).Return(errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			err := svc.DeleteRole(context.Background(), DeleteRoleInput{ID: roleID})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceListRoles 覆盖角色列表的成功与失败场景。
func TestServiceListRoles(t *testing.T) {
	cases := []struct {
		name      string
		prepare   func(*mockRepository)
		wantErr   bool
		wantCount int
	}{
		{
			name: "成功获取列表",
			prepare: func(m *mockRepository) {
				m.On("ListRoles", mock.Anything).Return([]Role{{Name: "ADMIN"}}, nil)
			},
			wantCount: 1,
		},
		{
			name:    "数据库错误",
			prepare: func(m *mockRepository) { m.On("ListRoles", mock.Anything).Return(nil, errors.New("db")) },
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			roles, err := svc.ListRoles(context.Background())
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, roles, tc.wantCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceCreatePermission 校验权限创建逻辑的输入校验与错误传播。
func TestServiceCreatePermission(t *testing.T) {
	cases := []struct {
		name    string
		input   CreatePermissionInput
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name:  "成功创建并小写化字段",
			input: CreatePermissionInput{Resource: " System ", Action: "VIEW", Description: "查看"},
			prepare: func(m *mockRepository) {
				m.On("CreatePermission", mock.Anything, mock.MatchedBy(func(p *Permission) bool {
					return p.Resource == "system" && p.Action == "view" && p.Description == "查看"
				})).Return(nil)
			},
		},
		{
			name:    "缺失字段",
			input:   CreatePermissionInput{Resource: "  ", Action: ""},
			prepare: func(m *mockRepository) {},
			wantErr: true,
		},
		{
			name:  "仓储错误",
			input: CreatePermissionInput{Resource: "system", Action: "edit"},
			prepare: func(m *mockRepository) {
				m.On("CreatePermission", mock.Anything, mock.AnythingOfType("*rbac.Permission")).Return(errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			permission, err := svc.CreatePermission(context.Background(), tc.input)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, permission)
			} else {
				require.NoError(t, err)
				require.NotNil(t, permission)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceUpdatePermission 使用表驱动覆盖权限更新逻辑。
func TestServiceUpdatePermission(t *testing.T) {
	newResource := " ACCOUNT "
	newAction := "LIST"
	newDesc := "列出账户"
	cases := []struct {
		name    string
		input   UpdatePermissionInput
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name: "成功更新所有字段",
			input: UpdatePermissionInput{
				ID:          uuid.New(),
				Resource:    &newResource,
				Action:      &newAction,
				Description: &newDesc,
			},
			prepare: func(m *mockRepository) {
				existing := &Permission{ID: uuid.New(), Resource: "system", Action: "view"}
				m.On("FindPermissionByID", mock.Anything, mock.Anything).Return(existing, nil)
				m.On("UpdatePermission", mock.Anything, mock.MatchedBy(func(p *Permission) bool {
					return p.Resource == "account" && p.Action == "list" && p.Description == "列出账户"
				})).Return(nil)
			},
		},
		{
			name:  "资源为空触发错误",
			input: UpdatePermissionInput{ID: uuid.New(), Resource: func() *string { v := " "; return &v }()},
			prepare: func(m *mockRepository) {
				existing := &Permission{ID: uuid.New(), Resource: "system", Action: "view"}
				m.On("FindPermissionByID", mock.Anything, mock.Anything).Return(existing, nil)
			},
			wantErr: true,
		},
		{
			name:  "查找失败",
			input: UpdatePermissionInput{ID: uuid.New()},
			prepare: func(m *mockRepository) {
				m.On("FindPermissionByID", mock.Anything, mock.Anything).Return(nil, errors.New("db"))
			},
			wantErr: true,
		},
		{
			name:  "保存失败",
			input: UpdatePermissionInput{ID: uuid.New()},
			prepare: func(m *mockRepository) {
				existing := &Permission{ID: uuid.New(), Resource: "system", Action: "view"}
				m.On("FindPermissionByID", mock.Anything, mock.Anything).Return(existing, nil)
				m.On("UpdatePermission", mock.Anything, mock.AnythingOfType("*rbac.Permission")).Return(errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			permission, err := svc.UpdatePermission(context.Background(), tc.input)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, permission)
			} else {
				require.NoError(t, err)
				require.NotNil(t, permission)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceDeletePermission 验证删除权限时的错误处理。
func TestServiceDeletePermission(t *testing.T) {
	permissionID := uuid.New()
	cases := []struct {
		name    string
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name: "删除成功",
			prepare: func(m *mockRepository) {
				m.On("DeletePermission", mock.Anything, permissionID).Return(nil)
			},
		},
		{
			name: "删除失败",
			prepare: func(m *mockRepository) {
				m.On("DeletePermission", mock.Anything, permissionID).Return(errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			err := svc.DeletePermission(context.Background(), DeletePermissionInput{ID: permissionID})
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceListPermissions 覆盖权限列表获取逻辑。
func TestServiceListPermissions(t *testing.T) {
	cases := []struct {
		name      string
		prepare   func(*mockRepository)
		wantErr   bool
		wantCount int
	}{
		{
			name: "返回数据",
			prepare: func(m *mockRepository) {
				m.On("ListPermissions", mock.Anything).Return([]Permission{{Resource: "system", Action: "view"}}, nil)
			},
			wantCount: 1,
		},
		{
			name:    "仓储错误",
			prepare: func(m *mockRepository) { m.On("ListPermissions", mock.Anything).Return(nil, errors.New("db")) },
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			permissions, err := svc.ListPermissions(context.Background())
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, permissions, tc.wantCount)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceAssignPermissions 覆盖权限分配给角色的完整分支。
func TestServiceAssignPermissions(t *testing.T) {
	roleID := uuid.New()
	validKeys := []string{"system:view", " invalid "}
	cases := []struct {
		name    string
		input   AssignRolePermissionsInput
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name:  "成功替换权限集合",
			input: AssignRolePermissionsInput{RoleID: roleID, Permissions: validKeys},
			prepare: func(m *mockRepository) {
				role := &Role{ID: roleID}
				perms := []*Permission{{ID: uuid.New(), Resource: "system", Action: "view"}}
				m.On("FindRoleByID", mock.Anything, roleID).Return(role, nil)
				m.On("FindPermissionsByKeys", mock.Anything, []string{"system:view"}).Return(perms, nil)
				m.On("ReplaceRolePermissions", mock.Anything, role, perms).Return(nil)
			},
		},
		{
			name:    "无效权限键",
			input:   AssignRolePermissionsInput{RoleID: roleID, Permissions: []string{"  "}},
			prepare: func(m *mockRepository) {},
			wantErr: true,
		},
		{
			name:  "角色不存在",
			input: AssignRolePermissionsInput{RoleID: roleID, Permissions: []string{"system:view"}},
			prepare: func(m *mockRepository) {
				m.On("FindRoleByID", mock.Anything, roleID).Return(nil, gorm.ErrRecordNotFound)
			},
			wantErr: true,
		},
		{
			name:  "权限不存在",
			input: AssignRolePermissionsInput{RoleID: roleID, Permissions: []string{"system:view"}},
			prepare: func(m *mockRepository) {
				role := &Role{ID: roleID}
				m.On("FindRoleByID", mock.Anything, roleID).Return(role, nil)
				m.On("FindPermissionsByKeys", mock.Anything, []string{"system:view"}).Return(nil, nil)
			},
			wantErr: true,
		},
		{
			name:  "替换失败",
			input: AssignRolePermissionsInput{RoleID: roleID, Permissions: []string{"system:view"}},
			prepare: func(m *mockRepository) {
				role := &Role{ID: roleID}
				perms := []*Permission{{ID: uuid.New(), Resource: "system", Action: "view"}}
				m.On("FindRoleByID", mock.Anything, roleID).Return(role, nil)
				m.On("FindPermissionsByKeys", mock.Anything, []string{"system:view"}).Return(perms, nil)
				m.On("ReplaceRolePermissions", mock.Anything, role, perms).Return(errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			role, err := svc.AssignPermissions(context.Background(), tc.input)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, role)
			} else {
				require.NoError(t, err)
				require.NotNil(t, role)
				require.Len(t, role.Permissions, 1)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceGetRolePermissions 验证角色权限读取逻辑。
func TestServiceGetRolePermissions(t *testing.T) {
	roleID := uuid.New()
	cases := []struct {
		name    string
		prepare func(*mockRepository)
		wantErr bool
		wantLen int
	}{
		{
			name: "返回权限集合",
			prepare: func(m *mockRepository) {
				role := &Role{ID: roleID, Permissions: []*Permission{{Resource: "system", Action: "view"}}}
				m.On("FindRoleByID", mock.Anything, roleID).Return(role, nil)
			},
			wantLen: 1,
		},
		{
			name:    "查询失败",
			prepare: func(m *mockRepository) { m.On("FindRoleByID", mock.Anything, roleID).Return(nil, errors.New("db")) },
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			keys, err := svc.GetRolePermissions(context.Background(), roleID)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, keys, tc.wantLen)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceGetRolesByNames 测试批量角色查询的去重与错误分支。
func TestServiceGetRolesByNames(t *testing.T) {
	cases := []struct {
		name    string
		input   []string
		prepare func(*mockRepository)
		wantErr bool
		wantLen int
	}{
		{
			name:  "成功获取并去重",
			input: []string{" admin ", "ADMIN", "user"},
			prepare: func(m *mockRepository) {
				roles := []*Role{{Name: "ADMIN"}, {Name: "USER"}}
				m.On("FindRolesByNames", mock.Anything, []string{"ADMIN", "USER"}).Return(roles, nil)
			},
			wantLen: 2,
		},
		{
			name:  "仓储错误",
			input: []string{"admin"},
			prepare: func(m *mockRepository) {
				m.On("FindRolesByNames", mock.Anything, []string{"ADMIN"}).Return(nil, errors.New("db"))
			},
			wantErr: true,
		},
		{
			name:  "角色缺失",
			input: []string{"admin", "user"},
			prepare: func(m *mockRepository) {
				roles := []*Role{{Name: "ADMIN"}}
				m.On("FindRolesByNames", mock.Anything, []string{"ADMIN", "USER"}).Return(roles, nil)
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			roles, err := svc.GetRolesByNames(context.Background(), tc.input)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, roles, tc.wantLen)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceEnsurePermissionsExist 测试批量确保权限存在的多种分支。
func TestServiceEnsurePermissionsExist(t *testing.T) {
	cases := []struct {
		name    string
		keys    []string
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name: "创建缺失权限",
			keys: []string{"system:view", "system:edit"},
			prepare: func(m *mockRepository) {
				existing := []*Permission{{Resource: "system", Action: "view"}}
				m.On("FindPermissionsByKeys", mock.Anything, []string{"system:view", "system:edit"}).Return(existing, nil)
				m.On("CreatePermission", mock.Anything, mock.MatchedBy(func(p *Permission) bool {
					return p.Resource == "system" && p.Action == "edit"
				})).Return(nil)
			},
		},
		{
			name: "查询失败",
			keys: []string{"system:view"},
			prepare: func(m *mockRepository) {
				m.On("FindPermissionsByKeys", mock.Anything, []string{"system:view"}).Return(nil, errors.New("db"))
			},
			wantErr: true,
		},
		{
			name:    "没有合法键跳过",
			keys:    []string{"  "},
			prepare: func(m *mockRepository) {},
		},
		{
			name: "忽略重复错误",
			keys: []string{"system:create"},
			prepare: func(m *mockRepository) {
				m.On("FindPermissionsByKeys", mock.Anything, []string{"system:create"}).Return(nil, nil)
				m.On("CreatePermission", mock.Anything, mock.AnythingOfType("*rbac.Permission")).Return(gorm.ErrDuplicatedKey)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			err := svc.EnsurePermissionsExist(context.Background(), tc.keys)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceEnsureAdminHasAllPermissions 校验管理员同步逻辑。
func TestServiceEnsureAdminHasAllPermissions(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(*mockRepository)
		wantErr bool
	}{
		{
			name: "成功同步",
			prepare: func(m *mockRepository) {
				adminRole := &Role{ID: uuid.New(), Name: NormalizeRoleName(constant.RoleAdmin)}
				userRole := &Role{ID: uuid.New(), Name: NormalizeRoleName(constant.RoleUser)}
				m.On("FindRoleByName", mock.Anything, NormalizeRoleName(constant.RoleAdmin)).Return(adminRole, nil).Twice()
				m.On("FindRoleByName", mock.Anything, NormalizeRoleName(constant.RoleUser)).Return(userRole, nil)
				perms := []Permission{{ID: uuid.New(), Resource: "system", Action: "view"}}
				m.On("ListPermissions", mock.Anything).Return(perms, nil)
				m.On("ReplaceRolePermissions", mock.Anything, adminRole, mock.MatchedBy(func(ps []*Permission) bool {
					return len(ps) == 1 && ps[0].Resource == "system"
				})).Return(nil)
			},
		},
		{
			name: "列权限失败",
			prepare: func(m *mockRepository) {
				adminRole := &Role{ID: uuid.New(), Name: NormalizeRoleName(constant.RoleAdmin)}
				userRole := &Role{ID: uuid.New(), Name: NormalizeRoleName(constant.RoleUser)}
				m.On("FindRoleByName", mock.Anything, NormalizeRoleName(constant.RoleAdmin)).Return(adminRole, nil).Twice()
				m.On("FindRoleByName", mock.Anything, NormalizeRoleName(constant.RoleUser)).Return(userRole, nil)
				m.On("ListPermissions", mock.Anything).Return(nil, errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			err := svc.EnsureAdminHasAllPermissions(context.Background())
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

// TestServiceHasPermission 确认权限检查会透传到底层仓储。
func TestServiceHasPermission(t *testing.T) {
	userID := uuid.New()
	cases := []struct {
		name    string
		prepare func(*mockRepository)
		wantErr bool
		allowed bool
	}{
		{
			name: "拥有权限",
			prepare: func(m *mockRepository) {
				m.On("UserHasPermission", mock.Anything, userID, "system:view").Return(true, nil)
			},
			allowed: true,
		},
		{
			name: "查询错误",
			prepare: func(m *mockRepository) {
				m.On("UserHasPermission", mock.Anything, userID, "system:view").Return(false, errors.New("db"))
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepository{}
			tc.prepare(mockRepo)
			svc := newMockService(mockRepo)

			allowed, err := svc.HasPermission(context.Background(), userID, "system:view")
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.allowed, allowed)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
