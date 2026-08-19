import type { RouteRule, RouteRuleInput } from '@/types';
import { RoutesService } from '../../bindings/NetRoute-Manager';

/**
 * 路由规则接口 —— 后端实装
 *
 * 通过 wails3 生成的 Service 绑定调用 Go 后端接口,数据由后端持久化
 * (用户文档目录/NetRoute-Manager/routes.json)。
 */

/** 获取全部路由规则 */
export async function fetchRoutes(): Promise<RouteRule[]> {
    // bindings 中 Go 切片类型为 `RouteRule[] | null`(nil 切片),此处归一化为空数组
    return RoutesService.ListRoutes().then((routes) => routes ?? []);
}

/** 新增一条路由规则,返回创建后的完整条目(后端默认 checked=true) */
export async function createRoute(input: RouteRuleInput): Promise<RouteRule> {
    return RoutesService.CreateRoute(input);
}

/** 更新一条路由规则,返回更新后的完整条目 */
export async function updateRoute(id: string, input: RouteRuleInput): Promise<RouteRule> {
    return RoutesService.UpdateRoute(id, input);
}

/** 删除一条路由规则 */
export async function deleteRoute(id: string): Promise<void> {
    return RoutesService.DeleteRoute(id);
}
