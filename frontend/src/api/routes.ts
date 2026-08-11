import type { RouteRule, RouteRuleInput } from '@/types';
import { CreateRoute, DeleteRoute, ListRoutes, UpdateRoute } from '../../wailsjs/go/main/App';

/**
 * 路由规则接口 —— 后端实装
 *
 * 通过 wailsjs 绑定调用 Go 后端接口,数据由后端持久化
 * (用户文档目录/NetRoute-Manager/routes.json)。
 */

/** 获取全部路由规则 */
export async function fetchRoutes(): Promise<RouteRule[]> {
    return ListRoutes();
}

/** 新增一条路由规则,返回创建后的完整条目(后端默认 checked=true) */
export async function createRoute(input: RouteRuleInput): Promise<RouteRule> {
    return CreateRoute(input);
}

/** 更新一条路由规则,返回更新后的完整条目 */
export async function updateRoute(id: string, input: RouteRuleInput): Promise<RouteRule> {
    return UpdateRoute(id, input);
}

/** 删除一条路由规则 */
export async function deleteRoute(id: string): Promise<void> {
    return DeleteRoute(id);
}
