import { useEffect, useRef, useState } from 'react';
import { Copy, Minus, Square, X } from 'lucide-react';
import { Window } from '@wailsio/runtime';

/**
 * 窗口控制按钮(最小化/最大化还原/关闭):frameless 模式下替代原生标题栏按钮。
 *
 * - 最小化:Window.Minimise
 * - 最大化/还原:Window.ToggleMaximise,图标状态无事件推送,点击后查询 Window.IsMaximised + resize 兜底
 * - 关闭:Window.Close → 触发后端 WindowClosing hook,将窗口隐藏到系统托盘而非退出应用
 *
 * 位于标题栏拖拽区内,必须显式 no-drag 保证按钮可点击。
 */
const noDrag = { '--wails-draggable': 'no-drag' } as React.CSSProperties;

export function WindowControls() {
    const [maximised, setMaximised] = useState(false);
    const syncTimer = useRef<number | null>(null);

    const syncMaximised = () => {
        void Window.IsMaximised().then(setMaximised);
    };

    useEffect(() => {
        syncMaximised();
        // 无窗口状态事件推送:resize 兜底同步(最大化/还原均触发 resize)
        window.addEventListener('resize', syncMaximised);
        return () => {
            window.removeEventListener('resize', syncMaximised);
            if (syncTimer.current !== null) window.clearTimeout(syncTimer.current);
        };
    }, []);

    const handleToggleMaximise = () => {
        void Window.ToggleMaximise();
        // toggle 为异步,短暂延迟后查询真实状态刷新图标;连点时重置计时器
        if (syncTimer.current !== null) window.clearTimeout(syncTimer.current);
        syncTimer.current = window.setTimeout(syncMaximised, 150);
    };

    return (
        <div
            style={noDrag}
            role="group"
            aria-label="窗口控制"
            className="flex items-stretch self-stretch"
        >
            <button
                type="button"
                title="最小化"
                aria-label="最小化"
                onClick={() => Window.Minimise()}
                className="flex w-11 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
                <Minus className="size-4" />
            </button>
            <button
                type="button"
                title={maximised ? '还原' : '最大化'}
                aria-label={maximised ? '还原' : '最大化'}
                onClick={handleToggleMaximise}
                className="flex w-11 items-center justify-center text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            >
                {maximised ? <Copy className="size-3.5" /> : <Square className="size-3" />}
            </button>
            <button
                type="button"
                title="关闭"
                aria-label="关闭"
                onClick={() => Window.Close()}
                className="flex w-11 items-center justify-center text-muted-foreground transition-colors hover:bg-destructive hover:text-white"
            >
                <X className="size-4" />
            </button>
        </div>
    );
}
