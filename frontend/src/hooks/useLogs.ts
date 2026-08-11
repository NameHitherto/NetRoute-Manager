import { useCallback, useEffect, useState } from 'react';
import { fetchLogs } from '@/api/system';
import { EventsOff, EventsOn } from '../../wailsjs/runtime/runtime';
import type { LogEntry, LogLevel } from '@/types';

const MAX_LOGS = 50;

/** 运行日志状态管理:加载历史 + 后端事件流 + 追加 + 清空 */
export function useLogs() {
    const [logs, setLogs] = useState<LogEntry[]>([]);

    useEffect(() => {
        let cancelled = false;
        fetchLogs().then((data) => {
            if (!cancelled) setLogs(data);
        });
        return () => {
            cancelled = true;
        };
    }, []);

    /** 订阅后端引擎日志事件(service:log):DNS 解析、路由增删等运行日志实时流入 */
    useEffect(() => {
        const off = EventsOn('service:log', (payload: LogEntry) => {
            setLogs((prev) => [payload, ...prev.slice(0, MAX_LOGS - 1)]);
        });
        return () => off();
    }, []);

    /** 追加一条本地日志(最新在前,保留最近 50 条) */
    const addLog = useCallback((type: LogLevel, text: string) => {
        const entry: LogEntry = {
            id: Date.now(),
            time: new Date().toLocaleTimeString(),
            type,
            text,
        };
        setLogs((prev) => [entry, ...prev.slice(0, MAX_LOGS - 1)]);
    }, []);

    /** 清空日志 */
    const clearLogs = useCallback(() => {
        setLogs([]);
    }, []);

    return { logs, addLog, clearLogs };
}
