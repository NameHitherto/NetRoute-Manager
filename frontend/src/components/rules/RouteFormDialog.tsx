import { useEffect, useState } from 'react';
import { Globe } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import type { RouteRule, RouteRuleInput } from '@/types';

interface RouteFormDialogProps {
    open: boolean;
    onOpenChange: (open: boolean) => void;
    /** 编辑中的条目;为 null 表示新建 */
    editingItem: RouteRule | null;
    /**
     * 提交回调。失败时应抛出错误(由上层负责提示),调用方会保持弹窗打开。
     */
    onSubmit: (input: RouteRuleInput) => void | Promise<void>;
}

/** 新建/编辑路由规则弹窗 */
export function RouteFormDialog({ open, onOpenChange, editingItem, onSubmit }: RouteFormDialogProps) {
    const [domain, setDomain] = useState('');
    const [port, setPort] = useState('443');
    const [alias, setAlias] = useState('');
    const [saving, setSaving] = useState(false);

    // 打开弹窗时按编辑态初始化表单
    useEffect(() => {
        if (!open) return;
        setDomain(editingItem?.domain ?? '');
        setPort(editingItem?.port ?? '443');
        setAlias(editingItem?.alias ?? '');
    }, [open, editingItem]);

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (saving || !domain.trim()) return;
        setSaving(true);
        try {
            await onSubmit({
                domain: domain.trim(),
                port: port.trim() || '80',
                alias: alias.trim(),
            });
            onOpenChange(false);
        } catch {
            // 保存失败:错误提示已由上层(如 App)处理,保持弹窗打开
        } finally {
            setSaving(false);
        }
    };

    return (
        <Dialog open={open} onOpenChange={onOpenChange}>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                        <Globe className="size-4 text-muted-foreground" />
                        {editingItem ? '编辑路由条目' : '新建路由规则'}
                    </DialogTitle>
                    <DialogDescription>
                        {editingItem ? `修改 ${editingItem.domain} 的路由配置` : '添加一条需要分流的路由规则'}
                    </DialogDescription>
                </DialogHeader>

                <form onSubmit={handleSubmit} className="space-y-4">
                    <div className="space-y-1.5">
                        <Label htmlFor="rule-domain">
                            域名地址 (FQDN) <span className="text-destructive">*</span>
                        </Label>
                        <Input
                            id="rule-domain"
                            type="text"
                            required
                            placeholder="例如: api.github.com"
                            value={domain}
                            onChange={(e) => setDomain(e.target.value)}
                            className="font-mono"
                        />
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor="rule-port">目标端口号</Label>
                        <Input
                            id="rule-port"
                            type="text"
                            placeholder="443"
                            value={port}
                            onChange={(e) => setPort(e.target.value)}
                            className="font-mono"
                        />
                    </div>

                    <div className="space-y-1.5">
                        <Label htmlFor="rule-alias">别名 / 备注</Label>
                        <Input
                            id="rule-alias"
                            type="text"
                            placeholder="例如: AI 接口专用分流"
                            value={alias}
                            onChange={(e) => setAlias(e.target.value)}
                        />
                    </div>

                    <DialogFooter>
                        <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                            取消
                        </Button>
                        <Button type="submit" disabled={saving}>
                            保存条目
                        </Button>
                    </DialogFooter>
                </form>
            </DialogContent>
        </Dialog>
    );
}
