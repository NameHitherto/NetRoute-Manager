import type { RouteRule, ServiceStartResult } from '@/types';
import { GetServiceStatus, StartService, StopService } from '../../wailsjs/go/main/App';

/**
 * 核心路由服务接口 —— 后端实装
 *
 * 通过 wailsjs 绑定调用 Go 后端:启动时后端对勾选规则做真实 DNS 解析,
 * 并将解析出的 IP 以临时主机路由绑定到所选网卡;DNS 服务器/解析间隔
 * 取自后端持久化设置,解析结果变化由后端 diff 同步路由表。
 */

/** 启动核心路由服务,返回服务状态快照(含已解析结果) */
export async function startService(params: {
    nicId: string;
    rules: RouteRule[];
}): Promise<ServiceStartResult> {
    return StartService(params.nicId, params.rules);
}

/** 停止核心路由服务(后端清理全部临时路由,恢复系统默认) */
export async function stopService(): Promise<void> {
    return StopService();
}

/** 获取当前服务状态快照(轮询刷新实时解析结果) */
export async function getServiceStatus(): Promise<ServiceStartResult> {
    return GetServiceStatus();
}
