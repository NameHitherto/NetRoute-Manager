import { useCallback, useState } from 'react';
import type { Theme } from '@/types';

/** 界面主题管理(浅色/深色),demo 默认深色 */
export function useTheme(initial: Theme = 'dark') {
    const [theme, setTheme] = useState<Theme>(initial);

    const toggleTheme = useCallback(() => {
        setTheme((prev) => (prev === 'dark' ? 'light' : 'dark'));
    }, []);

    return { theme, toggleTheme };
}
