import { useCallback, useEffect, useMemo, useState } from 'react';
import { createRoute, deleteRoute, fetchRoutes, updateRoute } from '@/api/routes';
import type { RouteRule, RouteRuleInput } from '@/types';

/**
 * 路由规则状态管理:加载列表 + CRUD + 勾选 + 搜索过滤。
 * 增删改均通过 api 层(mock)执行,后端实装后此处无需改动。
 */
export function useRoutes() {
    const [routes, setRoutes] = useState<RouteRule[]>([]);
    const [loading, setLoading] = useState(true);
    const [searchQuery, setSearchQuery] = useState('');

    useEffect(() => {
        let cancelled = false;
        fetchRoutes().then((data) => {
            if (cancelled) return;
            setRoutes(data);
            setLoading(false);
        });
        return () => {
            cancelled = true;
        };
    }, []);

    /** 新增规则,返回新条目 */
    const addRoute = useCallback(async (input: RouteRuleInput): Promise<RouteRule> => {
        const item = await createRoute(input);
        setRoutes((prev) => [item, ...prev]);
        return item;
    }, []);

    /** 更新规则 */
    const editRoute = useCallback(async (id: string, input: RouteRuleInput): Promise<void> => {
        const item = await updateRoute(id, input);
        setRoutes((prev) => prev.map((r) => (r.id === id ? item : r)));
    }, []);

    /** 删除规则 */
    const removeRoute = useCallback(async (id: string): Promise<void> => {
        await deleteRoute(id);
        setRoutes((prev) => prev.filter((r) => r.id !== id));
    }, []);

    /** 勾选/取消单条规则(服务运行中禁止) */
    const toggleCheck = useCallback((id: string) => {
        setRoutes((prev) => prev.map((r) => (r.id === id ? { ...r, checked: !r.checked } : r)));
    }, []);

    /** 全选/取消全选(服务运行中禁止) */
    const toggleAll = useCallback(() => {
        setRoutes((prev) => {
            const allChecked = prev.length > 0 && prev.every((r) => r.checked);
            return prev.map((r) => ({ ...r, checked: !allChecked }));
        });
    }, []);

    /** 按搜索词过滤后的列表(保留 demo 行为:运行中隐藏未勾选条目) */
    const visibleRoutes = useMemo(() => {
        return routes.filter((r) => {
            const q = searchQuery.trim().toLowerCase();
            if (q) {
                return r.domain.toLowerCase().includes(q) || r.alias.toLowerCase().includes(q) || r.port.includes(q);
            }
            return true;
        });
    }, [routes, searchQuery]);

    return {
        routes,
        setRoutes,
        loading,
        searchQuery,
        setSearchQuery,
        visibleRoutes,
        addRoute,
        editRoute,
        removeRoute,
        toggleCheck,
        toggleAll,
    };
}
