package request

// Pagination 定义了标准的分页请求参数
type Pagination struct {
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	OrderBy  string `json:"orderBy"`
}
