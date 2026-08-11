import { Edit3, Info, RefreshCw, Trash2 } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { cn } from '@/lib/utils';
import type { AppSettings, RouteRule } from '@/types';

interface RouteItemProps {
    item: RouteRule;
    isRunning: boolean;
    settings: AppSettings;
    onToggleCheck: (id: string) => void;
    onEdit: (item: RouteRule) => void;
    onDelete: (id: string) => void;
}

/** 单条路由规则(列表行):勾选 + 域名/端口/别名 + 运行态解析信息 + 操作按钮 */
export function RouteItem({ item, isRunning, settings, onToggleCheck, onEdit, onDelete }: RouteItemProps) {
    return (
        <div
            className={cn(
                'flex items-center justify-between p-3.5 transition-colors sm:p-4',
                item.checked && !isRunning ? 'bg-muted/50' : 'hover:bg-muted/30'
            )}
        >
            {/* 左侧:勾选框 + 域名 + 端口 + 别名 */}
            <div className="flex min-w-0 flex-1 items-center gap-3.5 pr-4">
                {!isRunning && (
                    <Checkbox
                        checked={item.checked}
                        onCheckedChange={() => onToggleCheck(item.id)}
                        aria-label={`勾选 ${item.domain}`}
                        className="border-border"
                    />
                )}

                <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                        <span className="font-mono text-sm font-semibold tracking-wide text-foreground">
                            {item.domain}
                        </span>
                        <Badge variant="outline" className="font-mono text-[11px]">
                            :{item.port}
                        </Badge>
                        {!isRunning && (
                            <Badge
                                variant={item.checked ? 'default' : 'secondary'}
                                className="font-mono text-[10px]"
                            >
                                {item.checked ? 'ENABLED' : 'DISABLED'}
                            </Badge>
                        )}
                    </div>

                    {item.alias && (
                        <p className="mt-1 flex items-center gap-1.5 truncate text-xs text-muted-foreground">
                            <Info className="inline size-3 shrink-0" />
                            {item.alias}
                        </p>
                    )}
                </div>
            </div>

            {/* 右侧:运行态 IP 与秒数 / 编辑删除按钮 */}
            <div className="flex shrink-0 items-center gap-3">
                {isRunning ? (
                    <>
                        <div className="flex flex-col items-end">
                            <div className="flex items-center gap-2">
                                <span className="font-mono text-[10px] uppercase text-muted-foreground">
                                    {settings.enableIpv6 ? 'IPv6' : 'IPv4'}
                                </span>
                                <Badge variant="outline" className="font-mono text-xs font-bold">
                                    {item.resolvedIp || 'RESOLVING...'}
                                </Badge>
                            </div>
                        </div>
                        <Badge
                            variant="outline"
                            className="flex items-center gap-1.5 font-mono text-xs text-muted-foreground"
                        >
                            <RefreshCw className="size-3 animate-spin" style={{ animationDuration: '8s' }} />
                            {item.lastResolvedSec}s ago
                        </Badge>
                    </>
                ) : (
                    <div className="flex items-center gap-1">
                        <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => onEdit(item)}
                            title="编辑条目"
                            aria-label={`编辑 ${item.domain}`}
                        >
                            <Edit3 className="size-4" />
                        </Button>
                        <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => onDelete(item.id)}
                            title="删除条目"
                            aria-label={`删除 ${item.domain}`}
                        >
                            <Trash2 className="size-4" />
                        </Button>
                    </div>
                )}
            </div>
        </div>
    );
}
