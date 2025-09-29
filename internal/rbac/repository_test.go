package rbac

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// userRole 用于构造 user_roles 关联表，满足 UserHasPermission 查询需求。
type userRole struct {
	UserID uuid.UUID `gorm:"type:uuid"`
	RoleID uuid.UUID `gorm:"type:uuid"`
}

func (userRole) TableName() string {
	return "user_role"
}

// setupTestDB 创建独立的内存数据库并初始化基础结构。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)

	repo := NewRepository(db)
	require.NoError(t, repo.Migrate(context.Background()))
	require.NoError(t, db.AutoMigrate(&userRole{}))
	return db
}

// runInTransaction 在事务中执行测试逻辑，并在结束后自动回滚。
func runInTransaction(t *testing.T, db *gorm.DB, fn func(ctx context.Context, repo *Repository, tx *gorm.DB)) {
	t.Helper()

	tx := db.Begin()
	require.NoError(t, tx.Error)

	t.Cleanup(func() {
		require.NoError(t, tx.Rollback().Error)
	})

	fn(context.Background(), NewRepository(tx), tx)
}

// TestRepositoryCreateRole 验证 CreateRole 能够持久化角色信息。
func TestRepositoryCreateRole(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		role := &Role{ID: uuid.New(), Name: "ADMIN", Description: "超级管理员"}
		require.NoError(t, repo.CreateRole(ctx, role))

		var stored Role
		require.NoError(t, tx.WithContext(ctx).First(&stored, "id = ?", role.ID).Error)
		require.Equal(t, role.Name, stored.Name)
		require.Equal(t, role.Description, stored.Description)
	})
}

// TestRepositoryUpdateRole 验证 UpdateRole 可以更新角色字段。
func TestRepositoryUpdateRole(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		original := &Role{ID: uuid.New(), Name: "USER", Description: "普通用户"}
		require.NoError(t, tx.WithContext(ctx).Create(original).Error)

		original.Description = "高级用户"
		require.NoError(t, repo.UpdateRole(ctx, original))

		var stored Role
		require.NoError(t, tx.WithContext(ctx).First(&stored, "id = ?", original.ID).Error)
		require.Equal(t, "高级用户", stored.Description)
	})
}

// TestRepositoryDeleteRole 确认 DeleteRole 会删除角色以及角色权限关系。
func TestRepositoryDeleteRole(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		perm := &Permission{ID: uuid.New(), Resource: "rbac", Action: "view"}
		role := &Role{ID: uuid.New(), Name: "AUDITOR"}
		require.NoError(t, tx.WithContext(ctx).Create(perm).Error)
		require.NoError(t, tx.WithContext(ctx).Create(role).Error)
		require.NoError(t, tx.WithContext(ctx).Model(role).Association("Permissions").Append(perm))

		require.NoError(t, repo.DeleteRole(ctx, role.ID))

		var count int64
		require.NoError(t, tx.WithContext(ctx).Model(&Role{}).Where("id = ?", role.ID).Count(&count).Error)
		require.Zero(t, count)

		var relCount int64
		require.NoError(t, tx.WithContext(ctx).Table("role_permissions").Where("role_id = ?", role.ID).Count(&relCount).Error)
		require.Zero(t, relCount)
	})
}

// TestRepositoryListRoles 校验 ListRoles 能够按时间顺序返回包含权限的角色集合。
func TestRepositoryListRoles(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		permissions := []*Permission{
			{ID: uuid.New(), Resource: "rbac", Action: "create"},
			{ID: uuid.New(), Resource: "rbac", Action: "delete"},
		}
		for _, p := range permissions {
			require.NoError(t, tx.WithContext(ctx).Create(p).Error)
		}

		roles := []*Role{
			{ID: uuid.New(), Name: "ADMIN", Permissions: []*Permission{permissions[0], permissions[1]}},
			{ID: uuid.New(), Name: "USER", Permissions: []*Permission{permissions[0]}},
		}
		for _, r := range roles {
			require.NoError(t, tx.WithContext(ctx).Create(r).Error)
			require.NoError(t, tx.WithContext(ctx).Model(r).Association("Permissions").Append(r.Permissions))
		}

		result, err := repo.ListRoles(ctx)
		require.NoError(t, err)
		require.Len(t, result, 2)
		require.NotEmpty(t, result[0].Permissions)
	})
}

// TestRepositoryFindRoleByID 验证可以通过主键精确查找角色。
func TestRepositoryFindRoleByID(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		role := &Role{ID: uuid.New(), Name: "OPERATOR"}
		require.NoError(t, tx.WithContext(ctx).Create(role).Error)

		stored, err := repo.FindRoleByID(ctx, role.ID)
		require.NoError(t, err)
		require.Equal(t, role.ID, stored.ID)
	})
}

// TestRepositoryFindRoleByName 验证可以通过名称查找角色。
func TestRepositoryFindRoleByName(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		role := &Role{ID: uuid.New(), Name: "SUPPORT"}
		require.NoError(t, tx.WithContext(ctx).Create(role).Error)

		stored, err := repo.FindRoleByName(ctx, role.Name)
		require.NoError(t, err)
		require.Equal(t, role.Name, stored.Name)
	})
}

// TestRepositoryFindRolesByNames 覆盖批量查询并保证输入正规化。
func TestRepositoryFindRolesByNames(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		roleA := &Role{ID: uuid.New(), Name: "ADMIN"}
		roleB := &Role{ID: uuid.New(), Name: "USER"}
		require.NoError(t, tx.WithContext(ctx).Create(roleA).Error)
		require.NoError(t, tx.WithContext(ctx).Create(roleB).Error)

		roles, err := repo.FindRolesByNames(ctx, []string{" admin ", "user", ""})
		require.NoError(t, err)
		require.Len(t, roles, 2)
	})
}

// TestRepositoryCreatePermission 验证 CreatePermission 能够写入权限。
func TestRepositoryCreatePermission(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		permission := &Permission{ID: uuid.New(), Resource: "system", Action: "list"}
		require.NoError(t, repo.CreatePermission(ctx, permission))

		var stored Permission
		require.NoError(t, tx.WithContext(ctx).First(&stored, "id = ?", permission.ID).Error)
		require.Equal(t, permission.Resource, stored.Resource)
	})
}

// TestRepositoryUpdatePermission 确保 UpdatePermission 会更新权限信息。
func TestRepositoryUpdatePermission(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		permission := &Permission{ID: uuid.New(), Resource: "system", Action: "view"}
		require.NoError(t, tx.WithContext(ctx).Create(permission).Error)

		permission.Description = "查看系统"
		require.NoError(t, repo.UpdatePermission(ctx, permission))

		var stored Permission
		require.NoError(t, tx.WithContext(ctx).First(&stored, "id = ?", permission.ID).Error)
		require.Equal(t, "查看系统", stored.Description)
	})
}

// TestRepositoryDeletePermission 验证 DeletePermission 会删除记录。
func TestRepositoryDeletePermission(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		permission := &Permission{ID: uuid.New(), Resource: "system", Action: "delete"}
		require.NoError(t, tx.WithContext(ctx).Create(permission).Error)

		require.NoError(t, repo.DeletePermission(ctx, permission.ID))

		var count int64
		require.NoError(t, tx.WithContext(ctx).Model(&Permission{}).Where("id = ?", permission.ID).Count(&count).Error)
		require.Zero(t, count)
	})
}

// TestRepositoryListPermissions 校验 ListPermissions 返回全部权限。
func TestRepositoryListPermissions(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		permissions := []Permission{
			{ID: uuid.New(), Resource: "system", Action: "create"},
			{ID: uuid.New(), Resource: "system", Action: "view"},
		}
		require.NoError(t, tx.WithContext(ctx).Create(&permissions).Error)

		result, err := repo.ListPermissions(ctx)
		require.NoError(t, err)
		require.Len(t, result, 2)
	})
}

// TestRepositoryFindPermissionByID 验证主键查询权限。
func TestRepositoryFindPermissionByID(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		permission := &Permission{ID: uuid.New(), Resource: "system", Action: "update"}
		require.NoError(t, tx.WithContext(ctx).Create(permission).Error)

		stored, err := repo.FindPermissionByID(ctx, permission.ID)
		require.NoError(t, err)
		require.Equal(t, permission.ID, stored.ID)
	})
}

// TestRepositoryFindPermissionsByKeys 验证通过组合键批量查找权限。
func TestRepositoryFindPermissionsByKeys(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		permissions := []Permission{
			{ID: uuid.New(), Resource: "system", Action: "create"},
			{ID: uuid.New(), Resource: "system", Action: "delete"},
		}
		require.NoError(t, tx.WithContext(ctx).Create(&permissions).Error)

		result, err := repo.FindPermissionsByKeys(ctx, []string{"system:create", "system:delete", "invalid"})
		require.NoError(t, err)
		require.Len(t, result, 2)
	})
}

// TestRepositoryReplaceRolePermissions 确认 ReplaceRolePermissions 可以重建关联。
func TestRepositoryReplaceRolePermissions(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		role := &Role{ID: uuid.New(), Name: "EDITOR"}
		oldPermission := &Permission{ID: uuid.New(), Resource: "article", Action: "edit"}
		newPermission := &Permission{ID: uuid.New(), Resource: "article", Action: "publish"}
		require.NoError(t, tx.WithContext(ctx).Create(role).Error)
		require.NoError(t, tx.WithContext(ctx).Create(oldPermission).Error)
		require.NoError(t, tx.WithContext(ctx).Create(newPermission).Error)
		require.NoError(t, tx.WithContext(ctx).Model(role).Association("Permissions").Append(oldPermission))

		require.NoError(t, repo.ReplaceRolePermissions(ctx, role, []*Permission{newPermission}))

		var relCount int64
		require.NoError(t, tx.WithContext(ctx).Table("role_permissions").Where("role_id = ? AND permission_id = ?", role.ID, newPermission.ID).Count(&relCount).Error)
		require.EqualValues(t, 1, relCount)
	})
}

// TestRepositoryUserHasPermission 覆盖用户权限判断逻辑。
func TestRepositoryUserHasPermission(t *testing.T) {
	db := setupTestDB(t)

	runInTransaction(t, db, func(ctx context.Context, repo *Repository, tx *gorm.DB) {
		userID := uuid.New()
		role := &Role{ID: uuid.New(), Name: "REVIEWER"}
		permission := &Permission{ID: uuid.New(), Resource: "article", Action: "approve"}
		require.NoError(t, tx.WithContext(ctx).Create(role).Error)
		require.NoError(t, tx.WithContext(ctx).Create(permission).Error)
		require.NoError(t, tx.WithContext(ctx).Model(role).Association("Permissions").Append(permission))
		require.NoError(t, tx.WithContext(ctx).Create(&userRole{UserID: userID, RoleID: role.ID}).Error)

		allowed, err := repo.UserHasPermission(ctx, userID, PermissionKey("article", "approve"))
		require.NoError(t, err)
		require.True(t, allowed)

		denied, err := repo.UserHasPermission(ctx, userID, "invalid")
		require.NoError(t, err)
		require.False(t, denied)
	})
}
