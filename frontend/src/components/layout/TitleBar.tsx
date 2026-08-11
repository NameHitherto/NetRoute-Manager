import { Moon, Sun, Zap } from 'lucide-react';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { cn } from '@/lib/utils';
import type { Theme } from '@/types';

interface TitleBarProps {
    theme: Theme;
    onToggleTheme: () => void;
}

/** 顶部标题栏:应用标识 + 版本 + 服务状态 + 主题切换 */
export function TitleBar({ theme, onToggleTheme }: TitleBarProps) {
    const dark = theme === 'dark';

    return (
        <header
            className={cn(
                'flex items-center justify-between border-b px-4 py-2 text-xs',
                dark ? 'bg-card border-border' : 'bg-card border-border'
            )}
        >
            <div className="flex items-center gap-2 font-medium">
                <div
                    className={cn(
                        'rounded-md border p-1',
                        dark ? 'border-border bg-muted text-foreground' : 'border-border bg-muted text-foreground'
                    )}
                >
                    <Zap className="size-3.5" />
                </div>
                <span className="font-semibold tracking-tight">WinRoute Minimal</span>
                <Badge variant="outline" className="font-mono text-[10px]">
                    v2.5.0
                </Badge>
            </div>

            <div className="flex items-center gap-3">
                {/* 主题切换:图标展示目标主题 */}
                <Button
                    variant="outline"
                    size="sm"
                    onClick={onToggleTheme}
                    title={dark ? '切换为亮色极简主题' : '切换为暗色极简主题'}
                    className="gap-1.5 text-[11px]"
                >
                    {dark ? <Sun className="size-3.5" /> : <Moon className="size-3.5" />}
                    <span className="hidden sm:inline">{dark ? '日间' : '夜间'}</span>
                </Button>
            </div>
        </header>
    );
}
