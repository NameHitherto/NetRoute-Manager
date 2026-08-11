/**
 * 领域数据结构定义
 *
 * 这些类型对应后续后端接口的返回/入参契约。
 * 后端实现后,仅需调整 src/api 下的实现,类型定义保持不变。
 */

/** 路由规则条目 */
export interface RouteRule {
    id: string;
    domain: string;
    port: string;
    alias: string;
    checked: boolean;
    resolvedIp: string;
    lastResolvedSec: number;
}

/** 新建/编辑路由规则时提交的表单数据 */
export interface RouteRuleInput {
    domain: string;
    port: string;
    alias: string;
}

/** 物理网卡 */
export interface NetworkInterface {
    id: string;
    name: string;
    type: 'wired' | 'wireless';
    active: boolean;
}

/** DNS 解析模式 */
export type DnsMode = 'UDP' | 'DoH' | 'DoT';

/** 全局设置 */
export interface AppSettings {
    primaryDns: string;
    secondaryDns: string;
    queryInterval: number; // 秒
    enableIpv6: boolean;
    autoStart: boolean;
    minToTray: boolean;
    dnsMode: DnsMode;
}

/** 日志级别 */
export type LogLevel = 'info' | 'warn' | 'error' | 'success';

/** 运行日志条目 */
export interface LogEntry {
    id: number;
    time: string;
    type: LogLevel;
    text: string;
}

/** 视图 Tab 标识 */
export type TabKey = 'rules' | 'settings' | 'logs';

/** 界面主题 */
export type Theme = 'dark' | 'light';

/** 服务启动结果(当前为临时测试结构,后续由后端返回) */
export interface ServiceStartResult {
    running: boolean;
    nicId: string;
    /** 启动后已解析的路由规则快照 */
    rules: RouteRule[];
}
