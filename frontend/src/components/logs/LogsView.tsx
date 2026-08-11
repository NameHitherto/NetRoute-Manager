import { Terminal } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { ScrollArea } from '@/components/ui/scroll-area';
import { cn } from '@/lib/utils';
import type { LogEntry } from '@/types';

interface LogsViewProps {
    logs: LogEntry[];
    onClear: () => void;
}

/** 日志级别 → 文字颜色(终端风格,固定深色底) */
function logTextClass(type: LogEntry['type']): string {
    switch (type) {
        case 'error':
            return 'font-bold text-foreground';
        case 'warn':
            return 'text-muted-foreground';
        case 'success':
            return 'text-foreground';
        default:
            return 'text-muted-foreground/80';
    }
}

/** 运行日志视图:终端风格日志流 + 清空操作 */
export function LogsView({ logs, onClear }: LogsViewProps) {
    return (
        <div className="mx-auto flex h-full max-w-6xl flex-col">
            <div
                className={cn(
                    'flex flex-1 flex-col overflow-hidden rounded-xl border p-4 font-mono text-xs',
                    'border-border bg-zinc-900 text-zinc-100'
                )}
            >
                <div className="mb-3 flex items-center justify-between border-b border-zinc-800 pb-3">
                    <span className="flex items-center gap-2 text-xs text-zinc-400">
                        <Terminal className="size-4" />
                        系统底层路由注入日志
                    </span>
                    <Button
                        variant="outline"
                        size="xs"
                        onClick={onClear}
                        className="border-zinc-700 bg-zinc-800 text-[11px] text-zinc-300 hover:bg-zinc-700 hover:text-zinc-100"
                    >
                        清空日志
                    </Button>
                </div>

                <ScrollArea className="h-full flex-1 pr-2">
                    {logs.length === 0 ? (
                        <div className="py-8 text-center italic text-zinc-600">暂无系统输出日志</div>
                    ) : (
                        <div className="space-y-2">
                            {logs.map((log) => (
                                <div key={log.id} className="flex items-start gap-3">
                                    <span className="shrink-0 font-mono text-zinc-500">[{log.time}]</span>
                                    <span className={cn('break-all', logTextClass(log.type))}>{log.text}</span>
                                </div>
                            ))}
                        </div>
                    )}
                </ScrollArea>
            </div>
        </div>
    );
}
