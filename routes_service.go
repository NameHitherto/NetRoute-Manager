package main

import (
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"NetRoute-Manager/internal/models"
	"NetRoute-Manager/internal/store"
)

// RoutesService 路由规则 CRUD 服务,暴露给前端绑定调用。
type RoutesService struct {
	store *store.Store
}

// NewRoutesService 创建路由规则服务。
func NewRoutesService(s *store.Store) *RoutesService {
	return &RoutesService{store: s}
}

// ListRoutes 返回全部路由规则(持久化数据)。
func (s *RoutesService) ListRoutes() ([]models.RouteRule, error) {
	return s.store.LoadRoutes()
}

// CreateRoute 新建一条路由规则并持久化,返回创建后的完整条目。
// 与前端 mock 行为一致:新建条目默认勾选(checked=true)。
func (s *RoutesService) CreateRoute(input models.RouteRuleInput) (models.RouteRule, error) {
	if err := validateRouteInput(input); err != nil {
		return models.RouteRule{}, err
	}
	routes, err := s.store.LoadRoutes()
	if err != nil {
		return models.RouteRule{}, err
	}
	rule := models.RouteRule{
		ID:      uuid.NewString(),
		Domain:  strings.TrimSpace(input.Domain),
		Port:    strings.TrimSpace(input.Port),
		Alias:   strings.TrimSpace(input.Alias),
		Checked: true,
	}
	routes = append(routes, rule)
	if err := s.store.SaveRoutes(routes); err != nil {
		return models.RouteRule{}, err
	}
	return rule, nil
}

// UpdateRoute 更新一条已存在的路由规则(仅更新 domain/port/alias,
// 保留 checked/resolvedIp/lastResolvedSec 等运行期字段)。
func (s *RoutesService) UpdateRoute(id string, input models.RouteRuleInput) (models.RouteRule, error) {
	if err := validateRouteInput(input); err != nil {
		return models.RouteRule{}, err
	}
	routes, err := s.store.LoadRoutes()
	if err != nil {
		return models.RouteRule{}, err
	}
	for i := range routes {
		if routes[i].ID != id {
			continue
		}
		routes[i].Domain = strings.TrimSpace(input.Domain)
		routes[i].Port = strings.TrimSpace(input.Port)
		routes[i].Alias = strings.TrimSpace(input.Alias)
		if err := s.store.SaveRoutes(routes); err != nil {
			return models.RouteRule{}, err
		}
		return routes[i], nil
	}
	return models.RouteRule{}, ErrRouteNotFound
}

// DeleteRoute 删除一条路由规则。
func (s *RoutesService) DeleteRoute(id string) error {
	routes, err := s.store.LoadRoutes()
	if err != nil {
		return err
	}
	for i, r := range routes {
		if r.ID == id {
			routes = append(routes[:i], routes[i+1:]...)
			return s.store.SaveRoutes(routes)
		}
	}
	return ErrRouteNotFound
}

// ErrRouteNotFound 表示目标路由规则不存在。
var ErrRouteNotFound = errors.New("路由规则不存在")

// validateRouteInput 校验路由规则表单数据:
// domain 必填;port 若填写则必须为数字(0-65535)。
func validateRouteInput(input models.RouteRuleInput) error {
	if strings.TrimSpace(input.Domain) == "" {
		return errors.New("域名不能为空")
	}
	if port := strings.TrimSpace(input.Port); port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 0 || n > 65535 {
			return errors.New("端口必须为 0-65535 的数字")
		}
	}
	return nil
}
