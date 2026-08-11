import { Network, Play, RefreshCw, Square } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { cn } from '@/lib/utils';
import type { NetworkInterface } from '@/types';

interface ControlBarProps {
    nics: NetworkInterface[];
    selectedNic: string;
    onSelectNic: (id: string) => void;
    isRunning: boolean;
    onToggleService: () => void;
    /** 服务启停中是否忙碌(防重复点击) */
    busy?: boolean;
    /** 网卡重新检测中(禁用刷新按钮) */
    refreshing?: boolean;
    /** 主动重新检测网卡 */
    onRefreshNics: () => void;
}

/** 核心控制区:绑定网卡选择 + 启动/停止路由服务 */
export function ControlBar({ nics, selectedNic, onSelectNic, isRunning, onToggleService, busy, refreshing, onRefreshNics }: ControlBarProps) {
    return (
        <section className="border-b border-border bg-card p-4 sm:p-5">
            <div className="mx-auto flex max-w-6xl flex-col justify-between gap-4 md:flex-row md:items-center">
                {/* 网卡选择 */}
                <div className="max-w-xl flex-1">
                    <Label className="mb-1.5 flex items-center gap-1.5 text-xs font-semibold text-muted-foreground">
                        <Network className="size-3.5" />
                        选择绑定的物理网络接口
                    </Label>
                    <Select value={selectedNic} onValueChange={onSelectNic} disabled={isRunning}>
                        <SelectTrigger className="w-full font-mono">
                            <SelectValue placeholder="选择网卡" />
                        </SelectTrigger>
                        <SelectContent position="popper">
                            {nics.map((nic) => (
                                <SelectItem key={nic.id} value={nic.id} className="font-mono">
                                    {nic.name}
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                    <Button
                        variant="outline"
                        size="sm"
                        className="mt-1.5"
                        onClick={onRefreshNics}
                        disabled={refreshing || isRunning}
                        aria-label="重新检测网卡"
                    >
                        <RefreshCw className={cn('size-3.5', refreshing && 'animate-spin')} />
                        刷新网卡
                    </Button>
                </div>
                <div className="flex items-center gap-3">
                    <Button
                        size="lg"
                        onClick={onToggleService}
                        disabled={busy}
                        aria-label={isRunning ? '停止路由服务' : '启动自定义路由'}
                    >
                        {isRunning ? (
                            <>
                                <Square className="size-4 fill-current" />
                                停止路由服务
                            </>
                        ) : (
                            <>
                                <Play className="size-4 fill-current" />
                                启动自定义路由
                            </>
                        )}
                    </Button>
                </div>
            </div>
        </section>
    );
}
