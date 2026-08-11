import type { AppSettings, NetworkInterface, LogEntry } from '@/types';

/**
 * 系统基础数据接口 —— 临时 Mock 实现
 *
 * 说明:当前仅作数据结构定义与声明,返回临时测试数据。
 * 待后端接口实现后,将各函数实现替换为 wailsjs 调用即可,调用方无需改动。
 */

const delay = (ms = 200) => new Promise((resolve) => setTimeout(resolve, ms));

/** 默认全局设置(临时测试数据) */
export const DEFAULT_SETTINGS: AppSettings = {
    primaryDns: '223.5.5.5',
    secondaryDns: '114.114.114.114',
    queryInterval: 30,
    enableIpv6: false,
    autoStart: true,
    minToTray: true,
    dnsMode: 'UDP',
};

/** 模拟检测到的物理网卡列表(临时测试数据) */
const NETWORK_INTERFACES: NetworkInterface[] = [
    { id: 'eth0', name: '以太网 (Intel Ethernet Controller I225-V)', type: 'wired', speed: '2.5 Gbps', active: true },
    { id: 'wlan0', name: 'Wi-Fi 6E (Intel Wi-Fi 6E AX211 160MHz)', type: 'wireless', speed: '1.2 Gbps', active: true },
];

/** 初始运行日志(临时测试数据) */
const INITIAL_LOGS: LogEntry[] = [
    { id: 1, time: '10:00:12', type: 'info', text: '网络适配器模块初始化完成,黑白极简界面就绪' },
];

/** 获取全局设置 */
export async function fetchSettings(): Promise<AppSettings> {
    await delay();
    return { ...DEFAULT_SETTINGS };
}

/** 保存全局设置 */
export async function saveSettings(settings: AppSettings): Promise<void> {
    await delay(100);
    void settings;
}

/** 获取物理网卡列表 */
export async function fetchNetworkInterfaces(): Promise<NetworkInterface[]> {
    await delay();
    return NETWORK_INTERFACES.map((nic) => ({ ...nic }));
}

/** 获取运行日志 */
export async function fetchLogs(): Promise<LogEntry[]> {
    await delay();
    return INITIAL_LOGS.map((log) => ({ ...log }));
}
