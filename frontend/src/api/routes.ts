import type { RouteRule, RouteRuleInput } from '@/types';

/**
 * 路由规则接口 —— 临时 Mock 实现
 *
 * 说明:当前仅作数据结构定义与声明,返回临时测试数据。
 * 待后端接口实现后,将本文件的实现替换为 wailsjs 调用
 * (如 import { ListRoutes } from '../../wailsjs/go/main/App') 即可,调用方无需改动。
 */

/** 模拟网络延迟,贴近真实接口的异步表现 */
const delay = (ms = 200) => new Promise((resolve) => setTimeout(resolve, ms));

/** 初始路由条目(临时测试数据) */
const INITIAL_ROUTES: RouteRule[] = [
    { id: '1', domain: 'api.openai.com', port: '443', alias: 'AI 服务 API 节点', checked: true, resolvedIp: '', lastResolvedSec: 0 },
    { id: '2', domain: 'github.com', port: '443', alias: 'GitHub 代码仓库主站', checked: true, resolvedIp: '', lastResolvedSec: 0 },
    { id: '3', domain: 'raw.githubusercontent.com', port: '443', alias: 'GitHub Raw 资源加速', checked: true, resolvedIp: '', lastResolvedSec: 0 },
    { id: '4', domain: 'steam-chat.valvesoftware.com', port: '27015', alias: 'Steam 客户端聊天节点', checked: false, resolvedIp: '', lastResolvedSec: 0 },
    { id: '5', domain: 'internal.corp.local', port: '8080', alias: '公司内网 VPN 分流站点', checked: false, resolvedIp: '', lastResolvedSec: 0 },
    { id: '6', domain: 'nflxso.net', port: '443', alias: '流媒体 CDN 分发', checked: true, resolvedIp: '', lastResolvedSec: 0 },
];

/** 获取全部路由规则 */
export async function fetchRoutes(): Promise<RouteRule[]> {
    await delay();
    return INITIAL_ROUTES.map((r) => ({ ...r }));
}

/** 新增一条路由规则,返回创建后的完整条目 */
export async function createRoute(input: RouteRuleInput): Promise<RouteRule> {
    await delay(100);
    return {
        id: String(Date.now()),
        domain: input.domain,
        port: input.port,
        alias: input.alias,
        checked: true,
        resolvedIp: '',
        lastResolvedSec: 0,
    };
}

/** 更新一条路由规则,返回更新后的完整条目 */
export async function updateRoute(id: string, input: RouteRuleInput): Promise<RouteRule> {
    await delay(100);
    return {
        id,
        domain: input.domain,
        port: input.port,
        alias: input.alias,
        checked: true,
        resolvedIp: '',
        lastResolvedSec: 0,
    };
}

/** 删除一条路由规则 */
export async function deleteRoute(id: string): Promise<void> {
    await delay(100);
    void id;
}
