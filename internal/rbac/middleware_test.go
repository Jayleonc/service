package rbac

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Jayleonc/service/internal/feature"
	"github.com/Jayleonc/service/pkg/constant"
	"github.com/Jayleonc/service/pkg/ginx/response"
)

// mockPermissionChecker 模拟权限校验器行为。
type mockPermissionChecker struct {
	mock.Mock
}

func (m *mockPermissionChecker) HasPermission(ctx context.Context, userID uuid.UUID, permission string) (bool, error) {
	args := m.Called(ctx, userID, permission)
	allowed, _ := args.Get(0).(bool)
	return allowed, args.Error(1)
}

// TestPermissionMiddleware 验证权限中间件的三种关键场景。
func TestPermissionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	permissionKey := "system:view"

	cases := []struct {
		name       string
		session    *feature.AuthContext
		prepare    func(*mockPermissionChecker)
		wantStatus int
		expectCall bool
	}{
		{
			name: "普通用户拥有权限",
			session: &feature.AuthContext{
				UserID: uuid.New(),
				Roles:  []string{"user"},
			},
			prepare: func(m *mockPermissionChecker) {
				m.On("HasPermission", mock.Anything, mock.Anything, permissionKey).Return(true, nil)
			},
			wantStatus: http.StatusOK,
			expectCall: true,
		},
		{
			name: "权限不足被拒绝",
			session: &feature.AuthContext{
				UserID: uuid.New(),
				Roles:  []string{"user"},
			},
			prepare: func(m *mockPermissionChecker) {
				m.On("HasPermission", mock.Anything, mock.Anything, permissionKey).Return(false, nil)
			},
			wantStatus: http.StatusForbidden,
			expectCall: true,
		},
		{
			name: "管理员直接放行",
			session: &feature.AuthContext{
				UserID: uuid.New(),
				Roles:  []string{constant.RoleAdmin},
			},
			prepare:    func(m *mockPermissionChecker) {},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			checker := &mockPermissionChecker{}
			tc.prepare(checker)

			middlewareFactory := NewPermissionMiddleware(checker)
			require.NotNil(t, middlewareFactory)

			router := gin.New()
			router.Use(func(c *gin.Context) {
				if tc.session != nil {
					feature.SetAuthContext(c, *tc.session)
				}
			})
			router.GET("/protected", middlewareFactory(permissionKey), func(c *gin.Context) {
				response.Success(c, gin.H{"ok": true})
			})

			req, err := http.NewRequest(http.MethodGet, "/protected", nil)
			require.NoError(t, err)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)

			require.Equal(t, tc.wantStatus, recorder.Code)
			if tc.expectCall {
				checker.AssertCalled(t, "HasPermission", mock.Anything, tc.session.UserID, permissionKey)
			} else {
				checker.AssertNotCalled(t, "HasPermission", mock.Anything, mock.Anything, permissionKey)
			}
		})
	}
}
