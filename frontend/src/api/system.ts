import type { AppSettings, NetworkInterface, LogEntry } from '@/types';
import { GetNetworkInterfaces, GetSettings, SaveSettings } from '../../wailsjs/go/main/App';

/**
 * 系统基础数据接口
 *
 * 设置读写与物理网卡检测已切换为 wailsjs 调用 Go 后端;
 * 运行日志暂未实装,保留临时 mock。
 */

const delay = (ms = 200) => new Promise((resolve) => setTimeout(resolve, ms));

/** 默认全局设置(后端首载默认值,仅作前端初始态兜底) */
export const DEFAULT_SETTINGS: AppSettings = {
    primaryDns: '223.5.5.5',
    secondaryDns: '114.114.114.114',
    queryInterval: 30,
    enableIpv6: false,
    autoStart: true,
    minToTray: true,
    dnsMode: 'UDP',
};

/** 初始运行日志(临时测试数据) */
const INITIAL_LOGS: LogEntry[] = [
    { id: 1, time: '10:00:12', type: 'info', text: '网络适配器模块初始化完成,黑白极简界面就绪' },
];

/** 获取全局设置 */
export async function fetchSettings(): Promise<AppSettings> {
    // wailsjs 生成的模型中 dnsMode 为 string,后端校验保证其为合法枚举值,此处断言收敛为联合类型
    return (await GetSettings()) as AppSettings;
}

/** 保存全局设置 */
export async function saveSettings(settings: AppSettings): Promise<void> {
    return SaveSettings(settings);
}

/** 获取本机活动物理网卡列表(后端 GetAdaptersAddresses 枚举) */
export async function fetchNetworkInterfaces(): Promise<NetworkInterface[]> {
    // wailsjs 生成的模型中 type 为 string,后端保证为 wired/wireless,此处断言收敛
    return (await GetNetworkInterfaces()) as NetworkInterface[];
}

/** 获取运行日志(临时 mock) */
export async function fetchLogs(): Promise<LogEntry[]> {
    await delay();
    return INITIAL_LOGS.map((log) => ({ ...log }));
}
