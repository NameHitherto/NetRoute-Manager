import { Layers, Search, Settings, Terminal, X } from 'lucide-react';
import { Input } from '@/components/ui/input';
import { TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import type { TabKey } from '@/types';

interface TabNavProps {
    /** 当前激活的 Tab(用于搜索框显隐) */
    activeTab: TabKey;
    /** 规则计数:运行中显示已生效条数,否则显示全部条数 */
    rulesCount: number;
    /** 搜索框(仅在规则视图显示) */
    searchQuery: string;
    onSearchChange: (value: string) => void;
    onClearSearch: () => void;
}

/** 导航 Tabs 栏 + 搜索框(需置于受控 Tabs 组件内使用) */
export function TabNav({
    activeTab,
    rulesCount,
    searchQuery,
    onSearchChange,
    onClearSearch,
}: TabNavProps) {
    return (
        <div className="flex items-center justify-between border-b border-border px-4">
            <TabsList variant="line" className="h-auto gap-1 rounded-none p-0">
                <TabsTrigger value="rules" className="gap-2 px-4 py-3">
                    <Layers className="size-4" />
                    路由规则列表
                    <span
                        className={cn(
                            'ml-1 rounded border px-1.5 py-0.5 font-mono text-[10px]',
                            activeTab === 'rules'
                                ? 'border-border bg-muted text-foreground'
                                : 'border-border bg-muted text-muted-foreground'
                        )}
                    >
                        {rulesCount}
                    </span>
                </TabsTrigger>
                <TabsTrigger value="settings" className="gap-2 px-4 py-3">
                    <Settings className="size-4" />
                    全局 DNS 配置
                </TabsTrigger>
                <TabsTrigger value="logs" className="gap-2 px-4 py-3">
                    <Terminal className="size-4" />
                    运行日志
                </TabsTrigger>
            </TabsList>

            {/* 搜索框:仅规则视图显示 */}
            {activeTab === 'rules' && (
                <div className="relative hidden w-60 sm:block">
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
            )}
        </div>
    );
}
