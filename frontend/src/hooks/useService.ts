import { useCallback, useEffect, useState, type Dispatch, type SetStateAction } from 'react';
import { mockResolveIp, startService, stopService } from '@/api/service';
import type { AppSettings, LogLevel, RouteRule } from '@/types';

interface UseServiceOptions {
    routes: RouteRule[];
    setRoutes: Dispatch<SetStateAction<RouteRule[]>>;
    settings: AppSettings;
    addLog: (type: LogLevel, text: string) => void;
    /** 无勾选规则时的提示回调(UI 层注入,避免 hook 内依赖 UI) */
    onNoActiveRules?: () => void;
}

/**
 * 核心路由服务状态管理:启停服务 + 每秒轮询刷新解析地址。
 * 轮询与 IP 生成当前由 mock 驱动,后端实装后替换为服务端推送即可。
 */
export function useService({ routes, setRoutes, settings, addLog, onNoActiveRules }: UseServiceOptions) {
    const [isRunning, setIsRunning] = useState(false);
    const [busy, setBusy] = useState(false);
    const [selectedNic, setSelectedNic] = useState('eth0');

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
                const result = await startService({
                    nicId: selectedNic,
                    rules: routes,
                    enableIpv6: settings.enableIpv6,
                });
                setRoutes(result.rules);
                setIsRunning(true);
                addLog('success', `核心路由服务已启动 [网卡: ${selectedNic}],共生效 ${activeCount} 条规则`);
            } finally {
                setBusy(false);
            }
        } else {
            setBusy(true);
            try {
                await stopService();
                setIsRunning(false);
                addLog('warn', '核心路由服务已手动停止');
            } finally {
                setBusy(false);
            }
        }
    }, [busy, isRunning, routes, selectedNic, settings.enableIpv6, setRoutes, addLog, onNoActiveRules]);

    /** 运行中每秒轮询:到达刷新间隔则重新解析,否则累计秒数 */
    useEffect(() => {
        if (!isRunning) return;
        const interval = setInterval(() => {
            setRoutes((prev) =>
                prev.map((item) => {
                    if (!item.checked) return item;
                    if (item.lastResolvedSec >= settings.queryInterval) {
                        return { ...item, resolvedIp: mockResolveIp(settings.enableIpv6), lastResolvedSec: 0 };
                    }
                    return { ...item, lastResolvedSec: item.lastResolvedSec + 1 };
                })
            );
        }, 1000);
        return () => clearInterval(interval);
    }, [isRunning, settings.queryInterval, settings.enableIpv6, setRoutes]);

    return { isRunning, busy, toggleService, selectedNic, setSelectedNic };
}
