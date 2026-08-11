import type { RouteRule, ServiceStartResult } from '@/types';

/**
 * 核心路由服务接口 —— 临时 Mock 实现
 *
 * 说明:当前仅作数据结构定义与声明,返回临时测试数据。
 * 待后端接口实现后,将各函数实现替换为 wailsjs 调用
 * (如 import { StartService, StopService } from '../../wailsjs/go/main/App') 即可,调用方无需改动。
 */

const delay = (ms = 200) => new Promise((resolve) => setTimeout(resolve, ms));

/** 随机生成真实感 IP 地址(临时测试数据,后端实现后移除) */
export function mockResolveIp(enableIpv6: boolean): string {
    if (enableIpv6) {
        return `2606:4700:${Math.floor(Math.random() * 8999 + 1000)}::${Math.floor(Math.random() * 89 + 10)}`;
    }
    return `${Math.floor(Math.random() * 150 + 104)}.${Math.floor(Math.random() * 200)}.${Math.floor(Math.random() * 250)}.${Math.floor(Math.random() * 250 + 1)}`;
}

/** 启动核心路由服务:为已勾选规则生成解析地址,返回服务状态快照 */
export async function startService(params: {
    nicId: string;
    rules: RouteRule[];
    enableIpv6: boolean;
}): Promise<ServiceStartResult> {
    await delay();
    const rules = params.rules.map((r) =>
        r.checked
            ? { ...r, resolvedIp: mockResolveIp(params.enableIpv6), lastResolvedSec: Math.floor(Math.random() * 5) }
            : r
    );
    return { running: true, nicId: params.nicId, rules };
}

/** 停止核心路由服务 */
export async function stopService(): Promise<void> {
    await delay(100);
}

/** 模拟一次 DNS 重新解析(轮询刷新用,后端实现后由服务端驱动) */
export async function reResolveRule(rule: RouteRule, enableIpv6: boolean): Promise<RouteRule> {
    await delay(80);
    return { ...rule, resolvedIp: mockResolveIp(enableIpv6), lastResolvedSec: 0 };
}
