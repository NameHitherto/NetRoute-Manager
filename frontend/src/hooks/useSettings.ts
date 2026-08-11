import { useCallback, useEffect, useState } from 'react';
import { DEFAULT_SETTINGS, fetchSettings } from '@/api/system';
import type { AppSettings } from '@/types';

/** 全局设置状态管理:加载 + 局部更新 + 快捷预设(后端实装后由接口读写) */
export function useSettings() {
    const [settings, setSettings] = useState<AppSettings>(DEFAULT_SETTINGS);

    useEffect(() => {
        let cancelled = false;
        fetchSettings().then((data) => {
            if (!cancelled) setSettings(data);
        });
        return () => {
            cancelled = true;
        };
    }, []);

    /** 局部更新设置 */
    const updateSettings = useCallback((patch: Partial<AppSettings>) => {
        setSettings((prev) => ({ ...prev, ...patch }));
    }, []);

    /** 应用快捷 DNS 预设 */
    const applyDnsPreset = useCallback(
        (primary: string, secondary: string) => {
            updateSettings({ primaryDns: primary, secondaryDns: secondary });
        },
        [updateSettings]
    );

    return { settings, updateSettings, applyDnsPreset };
}
