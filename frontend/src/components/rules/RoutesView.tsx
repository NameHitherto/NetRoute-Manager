import { Clock, Globe, Plus, ShieldCheck } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import { RouteItem } from '@/components/rules/RouteItem';
import type { AppSettings, RouteRule } from '@/types';

interface RoutesViewProps {
    routes: RouteRule[];
    visibleRoutes: RouteRule[];
    isRunning: boolean;
    settings: AppSettings;
    loading: boolean;
    onToggleAll: () => void;
    onToggleCheck: (id: string) => void;
    onDelete: (id: string) => void;
    onAdd: () => void;
    onEdit: (item: RouteRule) => void;
}

/** 路由规则管理视图:工具栏 / 运行提示 / 条目列表 / 空状态 */
export function RoutesView({
    routes,
    visibleRoutes,
    isRunning,
    settings,
    loading,
    onToggleAll,
    onToggleCheck,
    onDelete,
    onAdd,
    onEdit,
}: RoutesViewProps) {
    const checkedCount = routes.filter((r) => r.checked).length;
    const allChecked = routes.length > 0 && routes.every((r) => r.checked);

    return (
        <div className="mx-auto max-w-6xl space-y-4">
            {/* 未运行:操作工具栏 / 运行中:监控提示条 */}
            {!isRunning ? (
                <div className="flex items-center justify-between gap-3 rounded-lg border border-border bg-card p-3">
                    <div className="flex items-center gap-3">
                        <Button variant="outline" size="sm" onClick={onToggleAll}>
                            {allChecked ? '取消全选' : '全部勾选'}
                        </Button>
                        <span className="text-xs text-muted-foreground">
                            已勾选{' '}
                            <span className="font-mono font-semibold text-foreground">{checkedCount}</span> /{' '}
                            {routes.length} 条规则
                        </span>
                    </div>

                    <Button size="sm" onClick={onAdd}>
                        <Plus className="size-3.5" />
                        添加规则
                    </Button>
                </div>
            ) : (
                <div className="flex items-center justify-between rounded-lg border border-border bg-card p-3.5 font-mono text-xs">
                    <div className="flex items-center gap-2.5">
                        <ShieldCheck className="size-4 shrink-0" />
                        <span>
                            MONITORING_ACTIVE: 已自动屏蔽未勾选规则与编辑交互,实时响应 DNS 解析。
                        </span>
                    </div>
                    <Badge variant="outline" className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
                        <Clock className="size-3" />
                        {settings.queryInterval}s POLLING
                    </Badge>
                </div>
            )}

            {/* 条目列表 / 空状态 */}
            {visibleRoutes.length === 0 ? (
                <div
                    className={cn(
                        'rounded-xl border border-dashed py-16 text-center',
                        'border-border bg-card/40'
                    )}
                >
                    <Globe className="mx-auto mb-2 size-10 stroke-[1.5] text-muted-foreground" />
                    <p className="text-sm font-medium text-muted-foreground">暂无匹配的路由规则</p>
                    <p className="mt-1 text-xs text-muted-foreground/70">
                        {isRunning ? '没有包含已激活勾选的条目' : '点击右上角"添加规则"按钮添加新配置'}
                    </p>
                </div>
            ) : (
                <div className="overflow-hidden rounded-xl border border-border bg-card">
                    <div className="divide-y divide-border">
                        {visibleRoutes.map((item) => (
                            <RouteItem
                                key={item.id}
                                item={item}
                                isRunning={isRunning}
                                settings={settings}
                                onToggleCheck={onToggleCheck}
                                onEdit={onEdit}
                                onDelete={onDelete}
                            />
                        ))}
                    </div>
                </div>
            )}

            {loading && <p className="text-center text-xs text-muted-foreground">加载中...</p>}
        </div>
    );
}
