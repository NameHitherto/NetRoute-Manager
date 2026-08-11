import { useCallback, useEffect, useState } from 'react';
import { fetchLogs } from '@/api/system';
import type { LogEntry, LogLevel } from '@/types';

const MAX_LOGS = 50;

/** 运行日志状态管理:加载历史 + 追加(自动截断) + 清空 */
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

    /** 追加一条日志(最新在前,保留最近 50 条) */
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
