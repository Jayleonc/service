package request

// Pagination 定义了标准的分页请求参数
type Pagination struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	OrderBy  string `json:"order_by"`
}
