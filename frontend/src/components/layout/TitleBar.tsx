import { ArrowLeft, Moon, Plus, Search, Settings, Sun, X, Zap } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import type { Theme, ViewKey } from '@/types';

interface TitleBarProps {
    theme: Theme;
    onToggleTheme: () => void;
    /** 当前视图:首页(路由规则) / 设置 */
    view: ViewKey;
    onToggleSettings: () => void;
    /** 服务运行中禁止编辑(隐藏全选/添加规则按钮) */
    isRunning: boolean;
    /** 搜索框(仅首页显示) */
    searchQuery: string;
    onSearchChange: (value: string) => void;
    onClearSearch: () => void;
    /** 全选状态与操作(仅首页且服务未运行) */
    allChecked: boolean;
    onToggleAll: () => void;
    onAdd: () => void;
}

/** 顶部标题栏:应用标识 + 设置入口 + 首页操作(搜索/全选/添加规则) + 主题切换 */
export function TitleBar({
    theme,
    onToggleTheme,
    view,
    onToggleSettings,
    isRunning,
    searchQuery,
    onSearchChange,
    onClearSearch,
    allChecked,
    onToggleAll,
    onAdd,
}: TitleBarProps) {
    const dark = theme === 'dark';

    return (
        <header
            className={cn(
                'flex h-12 items-center justify-between gap-3 border-b px-4 text-xs',
                dark ? 'bg-card border-border' : 'bg-card border-border'
            )}
        >
            {/* 左:应用标识 + 设置入口(非首页时图标位置变为"返回"页头) */}
            <div className="flex min-w-0 items-center gap-2 font-medium">
                {view === 'settings' ? (
                    <Button
                        variant="ghost"
                        size="sm"
                        onClick={onToggleSettings}
                        aria-label="返回首页"
                        className="-ml-2 gap-1 px-2 text-[12px] text-muted-foreground hover:text-foreground"
                    >
                        <ArrowLeft className="size-4" />
                        返回
                    </Button>
                ) : (
                    <div
                        className={cn(
                            'rounded-md border p-1',
                            dark ? 'border-border bg-muted text-foreground' : 'border-border bg-muted text-foreground'
                        )}
                    >
                        <Zap className="size-3.5" />
                    </div>
                )}
                <span className="font-semibold tracking-tight">{view === 'settings' ? '设置' : 'WinRoute'}</span>
                <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={onToggleSettings}
                    title={view === 'settings' ? '返回路由规则' : '打开设置'}
                    aria-label="设置"
                    className={cn('ml-1 text-muted-foreground', view === 'settings' && 'bg-muted text-foreground')}
                >
                    <Settings className="size-4" />
                </Button>
            </div>

            {/* 右:首页操作(搜索/全选/添加规则) + 主题切换 */}
            <div className="flex items-center gap-2 sm:gap-3">
                {view === 'home' && (
                    <>
                        {/* 搜索框:仅首页显示 */}
                        <div className="relative hidden w-44 md:block">
                            <Search className="absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                            <Input
                                type="text"
                                placeholder="搜索域名 / 别名..."
                                value={searchQuery}
                                onChange={(e) => onSearchChange(e.target.value)}
                                className="pl-8 pr-8 text-xs"
                            />
                            {searchQuery && (
                                <button
                                    type="button"
                                    onClick={onClearSearch}
                                    className="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                                    aria-label="清空搜索"
                                >
                                    <X className="size-3" />
                                </button>
                            )}
                        </div>

                        {/* 全选 / 添加规则:仅首页且服务未运行 */}
                        {!isRunning && (
                            <>
                                <Button variant="outline" size="sm" onClick={onToggleAll} className="gap-1.5 text-[11px]">
                                    {allChecked ? '取消全选' : '全部勾选'}
                                </Button>
                                <Button size="sm" onClick={onAdd} className="gap-1 text-[11px]">
                                    <Plus className="size-3.5" />
                                    添加规则
                                </Button>
                            </>
                        )}
                    </>
                )}

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
