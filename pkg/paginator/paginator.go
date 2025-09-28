package paginator

import (
	"reflect"

	"gorm.io/gorm"

	"github.com/Jayleonc/service/pkg/request"
	"github.com/Jayleonc/service/pkg/response"
)

const (
	defaultPage     = 1
	defaultPageSize = 10
	maxPageSize     = 100
)

// Paginate 执行标准分页查询，返回统一分页结果
func Paginate(db *gorm.DB, req *request.Pagination, out any) (*response.PageResult, error) {
	if db == nil {
		return nil, gorm.ErrInvalidDB
	}

	page := defaultPage
	pageSize := defaultPageSize
	orderBy := ""

	if req != nil {
		if req.Page > 0 {
			page = req.Page
		}
		if req.PageSize > 0 {
			pageSize = req.PageSize
		}
		if req.OrderBy != "" {
			orderBy = req.OrderBy
		}
	}

	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	query := db.Session(&gorm.Session{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	if orderBy != "" {
		query = query.Order(orderBy)
	}

	if err := query.Offset(offset).Limit(pageSize).Find(out).Error; err != nil {
		return nil, err
	}

	list := out
	value := reflect.ValueOf(out)
	if value.Kind() == reflect.Ptr && !value.IsNil() {
		list = value.Elem().Interface()
	}

	return &response.PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
