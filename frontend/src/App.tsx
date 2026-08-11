import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { fetchNetworkInterfaces, saveSettings } from '@/api/system';
import { ControlBar } from '@/components/layout/ControlBar';
import { TabNav } from '@/components/layout/TabNav';
import { TitleBar } from '@/components/layout/TitleBar';
import { LogsView } from '@/components/logs/LogsView';
import { RouteFormDialog } from '@/components/rules/RouteFormDialog';
import { RoutesView } from '@/components/rules/RoutesView';
import { SettingsView } from '@/components/settings/SettingsView';
import { Toaster } from '@/components/ui/sonner';
import { Tabs, TabsContent } from '@/components/ui/tabs';
import { useLogs } from '@/hooks/useLogs';
import { useRoutes } from '@/hooks/useRoutes';
import { useService } from '@/hooks/useService';
import { useSettings } from '@/hooks/useSettings';
import { useTheme } from '@/hooks/useTheme';
import { cn } from '@/lib/utils';
import type { NetworkInterface, RouteRule, RouteRuleInput, TabKey } from '@/types';

function App() {
    // 主题
    const { theme, toggleTheme } = useTheme('light');

    // 领域状态(路由/设置数据来自后端持久化接口,网卡为临时 mock)
    const {
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
    } = useRoutes();
    const { settings, updateSettings, applyDnsPreset } = useSettings();
    const { logs, addLog, clearLogs } = useLogs();
    const { isRunning, busy, toggleService, selectedNic, setSelectedNic } = useService({
        routes,
        setRoutes,
        settings,
        addLog,
        onNoActiveRules: () => toast.error('请至少勾选一条需要生效的路由规则条目！'),
    });

    // 网卡列表(启动服务前预取)
    const [nics, setNics] = useState<NetworkInterface[]>([]);
    useEffect(() => {
        let cancelled = false;
        fetchNetworkInterfaces().then((data) => {
            if (!cancelled) setNics(data);
        });
        return () => {
            cancelled = true;
        };
    }, []);

    // 视图与弹窗状态
    const [activeTab, setActiveTab] = useState<TabKey>('rules');
    const [modalOpen, setModalOpen] = useState(false);
    const [editingItem, setEditingItem] = useState<RouteRule | null>(null);

    // 运行中仅展示已勾选条目(demo 行为),叠加搜索过滤
    const displayRoutes = useMemo(
        () => (isRunning ? visibleRoutes.filter((r) => r.checked) : visibleRoutes),
        [isRunning, visibleRoutes]
    );
    const rulesCount = isRunning ? displayRoutes.length : routes.length;

    const openAddModal = useCallback(() => {
        setEditingItem(null);
        setModalOpen(true);
    }, []);

    const openEditModal = useCallback((item: RouteRule) => {
        setEditingItem(item);
        setModalOpen(true);
    }, []);

    const handleSubmitRule = useCallback(
        async (input: RouteRuleInput) => {
            try {
                if (editingItem) {
                    await editRoute(editingItem.id, input);
                    addLog('info', `修改路由规则: ${input.domain}`);
                } else {
                    await addRoute(input);
                    addLog('info', `新增路由规则: ${input.domain}`);
                }
            } catch (error) {
                toast.error('保存路由规则失败,请稍后重试');
                // 失败时保持弹窗打开,由 RouteFormDialog 吞掉并复位保存状态
                throw error;
            }
        },
        [editingItem, editRoute, addRoute, addLog]
    );

    const handleDeleteRule = useCallback(
        async (id: string) => {
            try {
                await removeRoute(id);
                addLog('info', '删除了一条自定义路由条目');
            } catch {
                toast.error('删除路由规则失败,请稍后重试');
            }
        },
        [removeRoute, addLog]
    );

    /** 保存全局设置到后端持久化层 */
    const handleSaveSettings = useCallback(async () => {
        try {
            await saveSettings(settings);
            addLog('info', '全局设置已保存');
        } catch {
            toast.error('保存设置失败,请稍后重试');
        }
    }, [settings, addLog]);

    const dark = theme === 'dark';

    return (
        <div
            className={cn(
                'flex h-screen select-none flex-col overflow-hidden bg-background font-sans text-foreground antialiased',
                dark && 'dark'
            )}
        >
            <TitleBar theme={theme} onToggleTheme={toggleTheme}/>

            <ControlBar
                nics={nics}
                selectedNic={selectedNic}
                onSelectNic={setSelectedNic}
                isRunning={isRunning}
                busy={busy}
                onToggleService={() => void toggleService()}
            />

            <Tabs
                value={activeTab}
                onValueChange={(v) => setActiveTab(v as TabKey)}
                className="flex min-h-0 flex-1 flex-col gap-0"
            >
                <TabNav
                    activeTab={activeTab}
                    rulesCount={rulesCount}
                    searchQuery={searchQuery}
                    onSearchChange={setSearchQuery}
                    onClearSearch={() => setSearchQuery('')}
                />

                <TabsContent value="rules" className="min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
                    <RoutesView
                        routes={routes}
                        visibleRoutes={displayRoutes}
                        isRunning={isRunning}
                        settings={settings}
                        loading={loading}
                        onToggleAll={toggleAll}
                        onToggleCheck={toggleCheck}
                        onDelete={(id) => void handleDeleteRule(id)}
                        onAdd={openAddModal}
                        onEdit={openEditModal}
                    />
                </TabsContent>

                <TabsContent value="settings" className="min-h-0 flex-1 overflow-y-auto p-4 sm:p-6">
                    <SettingsView
                        settings={settings}
                        onUpdate={updateSettings}
                        onApplyDnsPreset={applyDnsPreset}
                        onSave={handleSaveSettings}
                    />
                </TabsContent>

                <TabsContent value="logs" className="flex min-h-0 flex-1 flex-col overflow-hidden p-4 sm:p-6">
                    <LogsView logs={logs} onClear={clearLogs} />
                </TabsContent>
            </Tabs>

            {/* 新建/编辑弹窗 */}
            <RouteFormDialog
                open={modalOpen}
                onOpenChange={setModalOpen}
                editingItem={editingItem}
                onSubmit={handleSubmitRule}
            />

            <Toaster theme={theme} position="bottom-right" />
        </div>
    );
}

export default App;
