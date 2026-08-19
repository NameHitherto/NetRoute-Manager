import type { RouteRule, ServiceStartResult } from '@/types';
import { EngineService } from '../../bindings/NetRoute-Manager';

/**
 * 核心路由服务接口 —— 后端实装
 *
 * 通过 wails3 生成的 Service 绑定调用 Go 后端:启动时后端对勾选规则做真实 DNS 解析,
 * 并将解析出的 IP 以临时主机路由绑定到所选网卡;DNS 服务器/解析间隔
 * 取自后端持久化设置,解析结果变化由后端 diff 同步路由表。
 */

/** bindings 中 Go 切片类型为 `T[] | null`(nil 切片),统一归一化为空数组 */
function normalizeStatus(result: Omit<ServiceStartResult, 'rules'> & { rules: RouteRule[] | null }): ServiceStartResult {
    return { ...result, rules: result.rules ?? [] };
}

/** 启动核心路由服务,返回服务状态快照(含已解析结果) */
export async function startService(params: {
    nicId: string;
    rules: RouteRule[];
}): Promise<ServiceStartResult> {
    return EngineService.StartService(params.nicId, params.rules).then(normalizeStatus);
}

/** 停止核心路由服务(后端清理全部临时路由,恢复系统默认) */
export async function stopService(): Promise<void> {
    return EngineService.StopService();
}

/** 获取当前服务状态快照(轮询刷新实时解析结果) */
export async function getServiceStatus(): Promise<ServiceStartResult> {
    return EngineService.GetServiceStatus().then(normalizeStatus);
}
