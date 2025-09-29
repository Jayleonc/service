package xerr

// 定义全局通用错误，错误码范围 1-999 预留给系统级错误使用。
var (
	ErrBadRequest     = New(400, "请求参数错误")
	ErrUnauthorized   = New(401, "未授权")
	ErrForbidden      = New(403, "禁止访问")
	ErrNotFound       = New(404, "资源未找到")
	ErrInternalServer = New(500, "服务器内部错误")
	ErrDatabase       = New(501, "数据库错误")
)
