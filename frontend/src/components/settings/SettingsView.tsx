import { useState } from 'react';
import { Server, Sliders } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import type { AppSettings } from '@/types';

/** 快捷 DNS 预设 */
const DNS_PRESETS = [
    { name: '阿里 DNS', p: '223.5.5.5', s: '223.6.6.6' },
    { name: '腾讯 DNSPod', p: '119.29.29.29', s: '1.12.12.12' },
    { name: 'Cloudflare', p: '1.1.1.1', s: '1.0.0.1' },
    { name: 'Google DNS', p: '8.8.8.8', s: '8.8.4.4' },
];

interface SettingsViewProps {
    settings: AppSettings;
    onUpdate: (patch: Partial<AppSettings>) => void;
    onApplyDnsPreset: (primary: string, secondary: string) => void;
    /** 持久化保存设置到后端(成功后由调用方反馈) */
    onSave: () => Promise<void>;
}

/** 全局设置视图:DNS 上游配置 + 高级网络协议 */
export function SettingsView({ settings, onUpdate, onApplyDnsPreset, onSave }: SettingsViewProps) {
    const [saving, setSaving] = useState(false);

    const handleSave = async () => {
        if (saving) return;
        setSaving(true);
        try {
            await onSave();
        } finally {
            setSaving(false);
        }
    };

    return (
        <div className="mx-auto max-w-4xl space-y-6">
            {/* DNS 服务器配置 */}
            <Card>
                <CardHeader className="border-b border-border">
                    <CardTitle className="flex items-center gap-2 text-sm">
                        <Server className="size-4 text-muted-foreground" />
                        DNS 服务器
                    </CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                        <div className="space-y-1.5">
                            <Label htmlFor="primary-dns" className="text-xs text-muted-foreground">
                                主 DNS 服务器
                            </Label>
                            <Input
                                id="primary-dns"
                                type="text"
                                value={settings.primaryDns}
                                onChange={(e) => onUpdate({ primaryDns: e.target.value })}
                                className="font-mono text-xs"
                            />
                        </div>
                        <div className="space-y-1.5">
                            <Label htmlFor="secondary-dns" className="text-xs text-muted-foreground">
                                备用 DNS 服务器
                            </Label>
                            <Input
                                id="secondary-dns"
                                type="text"
                                value={settings.secondaryDns}
                                onChange={(e) => onUpdate({ secondaryDns: e.target.value })}
                                className="font-mono text-xs"
                            />
                        </div>
                    </div>

                    {/* 快捷预设 */}
                    <div>
                        <span className="mb-1.5 block text-[11px] text-muted-foreground">
                            快捷导入预设 DNS:
                        </span>
                        <div className="flex flex-wrap gap-2">
                            {DNS_PRESETS.map((dns) => (
                                <Button
                                    key={dns.name}
                                    variant="outline"
                                    size="xs"
                                    className="font-mono text-[11px]"
                                    onClick={() => onApplyDnsPreset(dns.p, dns.s)}
                                >
                                    {dns.name} ({dns.p})
                                </Button>
                            ))}
                        </div>
                    </div>

                    {/* 解析轮询间隔:手动输入,单位为秒 */}
                    <div className="space-y-2 pt-2">
                        <Label htmlFor="query-interval" className="text-xs text-muted-foreground">
                            DNS 解析轮询刷新间隔
                        </Label>
                        <div className="flex items-center gap-2">
                            <Input
                                id="query-interval"
                                type="number"
                                min={5}
                                max={120}
                                step={5}
                                value={settings.queryInterval}
                                onChange={(e) => {
                                    // 留空时不写入,避免误存 0;合法数字直接更新
                                    if (e.target.value === '') return;
                                    const num = Number(e.target.value);
                                    if (Number.isFinite(num)) onUpdate({ queryInterval: num });
                                }}
                                className="w-28 font-mono text-xs"
                            />
                            <span className="text-xs text-muted-foreground">秒</span>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* 高级选项 */}
            <Card>
                <CardHeader className="border-b border-border">
                    <CardTitle className="flex items-center gap-2 text-sm">
                        <Sliders className="size-4 text-muted-foreground" />
                        高级网络协议
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="flex items-center justify-between py-1">
                        <div>
                            <div className="text-xs font-medium text-foreground">优先使用 IPv6 解析记录</div>
                            <div className="mt-0.5 text-[11px] text-muted-foreground">
                                开启后,域名路由解析将优先匹配 AAAA 格式记录
                            </div>
                        </div>
                        <Switch
                            checked={settings.enableIpv6}
                            onCheckedChange={(checked) => onUpdate({ enableIpv6: checked })}
                            aria-label="优先使用 IPv6 解析记录"
                        />
                    </div>
                </CardContent>
            </Card>

            {/* 保存操作区 */}
            <div className="flex justify-end">
                <Button onClick={() => void handleSave()} disabled={saving}>
                    {saving ? '保存中...' : '保存设置'}
                </Button>
            </div>
        </div>
    );
}
