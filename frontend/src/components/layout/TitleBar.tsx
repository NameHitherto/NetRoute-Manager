import { ArrowLeft, Moon, Plus, Search, Settings, Sun, X, Zap } from 'lucide-react';
import { WindowToggleMaximise } from '../../../wailsjs/runtime/runtime';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';
import { WindowControls } from '@/components/layout/WindowControls';
import type { Theme, ViewKey } from '@/types';

/** 标题栏为窗口拖拽区;交互元素必须显式覆盖为 no-drag(拖拽判定只看 e.target 的 CSS 变量,会继承) */
const drag = { '--wails-draggable': 'drag' } as React.CSSProperties;
const noDrag = { '--wails-draggable': 'no-drag' } as React.CSSProperties;

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

/** 顶部标题栏(左右两段):左侧应用标识(非首页为返回页头),右侧首页操作 + 设置 + 主题切换 + 窗口控制 */
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
            style={drag}
            onDoubleClick={(e) => {
                // 仅空白拖拽区双击触发最大化:判定与 Wails 运行时 dragTest 同源,
                // 交互元素(no-drag)及其内部节点双击均不会误触
                const target = e.target as HTMLElement;
                if (getComputedStyle(target).getPropertyValue('--wails-draggable') !== 'drag') return;
                void WindowToggleMaximise();
            }}
            className={cn(
                'flex h-12 items-center justify-between gap-3 border-b pl-4 text-xs',
                dark ? 'bg-card border-border' : 'bg-card border-border'
            )}
        >
            {/* 左:首页应用标识 / 非首页返回页头 */}
            <div className="flex min-w-0 items-center gap-2 font-medium">
                {view === 'settings' ? (
                    <>
                        <Button
                            variant="ghost"
                            size="sm"
                            onClick={onToggleSettings}
                            aria-label="返回首页"
                            style={noDrag}
                            className="-ml-2 gap-1 px-2 text-[12px] text-muted-foreground hover:text-foreground"
                        >
                            <ArrowLeft className="size-4" />
                            返回
                        </Button>
                        <span className="font-semibold tracking-tight">设置</span>
                    </>
                ) : (
                    <>
                        <div
                            className={cn(
                                'rounded-md border p-1',
                                dark ? 'border-border bg-muted text-foreground' : 'border-border bg-muted text-foreground'
                            )}
                        >
                            <Zap className="size-3.5" />
                        </div>
                        <span className="font-semibold tracking-tight">WinRoute</span>
                    </>
                )}
            </div>

            {/* 右:首页操作 + 设置 + 主题切换 + 窗口控制 */}
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
                                style={noDrag}
                                className="pl-8 pr-8 text-xs"
                            />
                            {searchQuery && (
                                <button
                                    type="button"
                                    onClick={onClearSearch}
                                    style={noDrag}
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
                                <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={onToggleAll}
                                    style={noDrag}
                                    className="gap-1.5 text-[11px]"
                                >
                                    {allChecked ? '取消全选' : '全部勾选'}
                                </Button>
                                <Button size="sm" onClick={onAdd} style={noDrag} className="gap-1 text-[11px]">
                                    <Plus className="size-3.5" />
                                    添加规则
                                </Button>
                            </>
                        )}
                    </>
                )}

                <div className="flex items-center">
                    {/* 设置:仅图标,点击切换首页/设置视图 */}
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={onToggleSettings}
                        style={noDrag}
                        className={cn('h-12 w-12 rounded-none text-muted-foreground', view === 'settings' && 'bg-muted text-foreground')}
                    >
                        <Settings className="size-4" />
                    </Button>

                    {/* 主题切换:仅图标,图标展示目标主题 */}
                    <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={onToggleTheme}
                        style={noDrag}
                        className="h-12 w-12 rounded-none text-muted-foreground"
                    >
                        {dark ? <Sun className="size-4" /> : <Moon className="size-4" />}
                    </Button>

                    {/* 窗口控制:最小化/最大化还原/关闭(frameless 替代原生标题栏按钮) */}
                    <WindowControls />
                </div>
            </div>
        </header>
    );
}
