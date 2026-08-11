import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from 'react';
import { getServiceStatus, startService, stopService } from '@/api/service';
import type { LogLevel, RouteRule } from '@/types';

interface UseServiceOptions {
    routes: RouteRule[];
    setRoutes: Dispatch<SetStateAction<RouteRule[]>>;
    addLog: (type: LogLevel, text: string) => void;
    /** 无勾选规则时的提示回调(UI 层注入,避免 hook 内依赖 UI) */
    onNoActiveRules?: () => void;
}

/**
 * 核心路由服务状态管理:启停服务 + 每秒轮询后端状态快照刷新解析结果。
 * 启停与 DNS 解析均在后端执行,前端仅负责快照同步与展示。
 */
export function useService({ routes, setRoutes, addLog, onNoActiveRules }: UseServiceOptions) {
    const [isRunning, setIsRunning] = useState(false);
    const [busy, setBusy] = useState(false);
    const [selectedNic, setSelectedNic] = useState('');

    /** 切换服务启停(忙碌期间防重复触发) */
    const toggleService = useCallback(async () => {
        if (busy) return;
        if (!isRunning) {
            const activeCount = routes.filter((r) => r.checked).length;
            if (activeCount === 0) {
                onNoActiveRules?.();
                return;
            }
            setBusy(true);
            try {
                const result = await startService({ nicId: selectedNic, rules: routes });
                setRoutes(result.rules);
                setIsRunning(true);
                addLog('success', `核心路由服务已启动 [网卡: ${selectedNic}],共生效 ${activeCount} 条规则`);
            } catch (error) {
                addLog('error', `核心路由服务启动失败: ${error instanceof Error ? error.message : String(error)}`);
            } finally {
                setBusy(false);
            }
        } else {
            setBusy(true);
            try {
                await stopService();
                setIsRunning(false);
                addLog('warn', '核心路由服务已手动停止,已清理全部临时路由');
            } catch (error) {
                addLog('error', `停止路由服务失败: ${error instanceof Error ? error.message : String(error)}`);
            } finally {
                setBusy(false);
            }
        }
    }, [busy, isRunning, routes, selectedNic, setRoutes, addLog, onNoActiveRules]);

    /** 运行中每秒轮询后端状态快照:刷新解析结果并累计距上次解析秒数 */
    useEffect(() => {
        if (!isRunning) return;
        const interval = setInterval(() => {
            void getServiceStatus()
                .then((status) => {
                    if (!status.running) {
                        // 后端已停止(如外部清理),同步前端状态
                        setIsRunning(false);
                        return;
                    }
                    const byId = new Map(status.rules.map((r) => [r.id, r]));
                    setRoutes((prev) =>
                        prev.map((item) => {
                            if (!item.checked) return item;
                            const fresh = byId.get(item.id);
                            if (!fresh) return item;
                            return {
                                ...item,
                                resolvedIp: fresh.resolvedIp,
                                lastResolvedSec: (item.lastResolvedSec ?? 0) + 1,
                            };
                        })
                    );
                })
                .catch(() => {
                    // 轮询失败静默,下轮重试
                });
        }, 1000);
        return () => clearInterval(interval);
    }, [isRunning, setRoutes]);

    return { isRunning, busy, toggleService, selectedNic, setSelectedNic };
}
