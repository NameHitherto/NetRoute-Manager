# NetRoute-Manager

[NetRoute-Policy-Manager](https://github.com/NameHitherto/NetRoute-Policy-Manager)（NetRoute VPS Host Route CLI）的 **GUI 图形化增强版本**。

原项目是一个面向 Windows 11 的小型 Go CLI：检测 v2rayN 当前代理服务器，并将 VPS 的 IP 以临时主机路由绑定到指定物理网卡，实现「为 VPS IP 指定特定网卡出口」的分流策略。本项目在此基础上提供完整的图形界面，将规则管理、网卡绑定、服务启停、实时解析状态与运行日志全部可视化，同时保留原项目的核心安全语义（临时路由、精确清理、崩溃恢复）。

## 功能特性

- **路由规则管理**：以列表方式维护「域名 / 端口 / 别名」规则，支持新增、编辑、删除、勾选启用与关键字搜索；服务运行中自动屏蔽编辑交互，仅展示已生效条目。
- **物理网卡绑定**：启动时自动枚举本机物理网卡并校验网络环境（无 IPv4 网关的网卡不可用），支持手动刷新网卡列表。
- **核心路由服务**：按设定的间隔定时对勾选规则做 DNS 解析，将解析出的 IP 以临时主机路由（IPv4 `/32`、IPv6 `/128`）绑定到选中网卡；解析结果变化时自动 diff 同步（新增建路由、消失删路由）。
- **崩溃恢复**：服务运行期间将已应用路由清单持久化，下次启动自动清理上次异常退出遗留的路由，保证系统路由表始终可恢复。
- **全局设置**：上游 DNS 服务器（主 / 备 + 阿里、腾讯 DNSPod、Cloudflare、Google 快捷预设）、解析轮询间隔（手动数字输入，单位秒）、优先使用 IPv6 解析记录。
- **运行日志**：终端风格的实时日志面板，展示服务启停、DNS 解析与路由注入事件，支持一键清空。
- **界面体验**：亮 / 暗主题切换、应用启动时自动请求管理员提权（路由操作需要）、自定义闪电图标。
- **系统托盘驻留**：点击标题栏关闭按钮时应用隐藏到系统托盘而非退出（路由服务继续运行），可通过托盘菜单「显示主窗口」恢复窗口、或「退出」彻底退出并清理临时路由。托盘基于 Wails v3 原生 `SystemTray` API 实现，无第三方依赖。
- **无边框自定义标题栏**：隐藏 Windows 原生标题栏（`Frameless`），由前端自绘标题栏提供最小化 / 最大化还原 / 关闭（隐藏到托盘）按钮，支持标题栏拖拽移动与双击最大化；窗口最小尺寸限制（800×600）依旧生效。注意：无边框模式下无法通过窗口边缘拖拽调整大小（Wails 框架限制），仅能通过最大化 / 还原切换尺寸。

## 与原 CLI 的增强对照

| NetRoute-Policy-Manager (CLI) | NetRoute-Manager (GUI) |
| --- | --- |
| `detect` / `apply` / `start` 子命令 | 图形化「启动自定义路由 / 停止路由服务」按钮 |
| `interfaces` 命令行枚举 | 网卡下拉选择 + 手动刷新按钮 |
| YAML 配置文件 | 应用内规则管理，JSON 本地持久化 |
| `status` 文本输出 | 列表实时轮询展示解析 IP 与距上次解析秒数 |
| 日志落盘 / 无界面反馈 | 终端风格实时运行日志面板 |
| `cleanup` 手动清理 | 停止服务 / 退出应用时自动清理临时路由 |

## 技术栈

- **桌面框架**：[Wails v3](https://v3.wails.io)（Go 后端 + Web 前端，原生 SystemTray / 单实例 / 事件系统）
- **后端**：Go（`internal/dns` DNS 解析、`internal/route` 主机路由注入、`internal/nics` 物理网卡识别、`internal/service` 路由服务引擎、`internal/store` JSON 持久化）
- **前端**：React 19 · TypeScript · Vite · Tailwind CSS 4 · shadcn/ui 风格组件 · lucide-react · radix-ui

## 快速开始（开发）

前置要求：Go、Node.js、Wails v3 CLI（`wails3`）。

```bash
# 前端依赖
cd frontend && npm install && cd ..

# 开发模式(前端热重载)
wails3 dev
```

仅调试前端时，可在 `frontend/` 下运行 `npm run dev`（后端接口不可用时界面为 mock 行为）。

## 构建

```bash
# 生产构建(frontend 自动构建,产物在 bin/NetRoute-Manager.exe)
wails3 build
```

仅构建前端：`cd frontend && npm run build`。前端 bindings 变更时重新生成：`wails3 generate bindings -ts`。

应用图标源文件位于 `build/appicon.png`（1024×1024），Windows exe 图标为 `build/windows/icon.ico`（含 16–256 全部尺寸），均为闪电（lucide `zap`）风格。

## 使用流程

1. 启动应用（自动请求管理员权限）。
2. 打开「设置」：配置上游 DNS、解析轮询间隔、是否优先 IPv6。
3. 添加并勾选需要生效的路由规则（至少一条）。
4. 在核心控制栏选择绑定的物理网卡，点击「启动自定义路由」。
5. 运行中可点击「运行日志」查看实时日志，列表内实时展示各规则的解析 IP。
6. 点击「停止路由服务」或通过系统托盘菜单「退出」彻底退出应用，临时路由自动清理，恢复系统默认选路。点击标题栏关闭按钮仅将应用隐藏到系统托盘，不会退出（可在托盘菜单「显示主窗口」恢复）。

## 安全语义

- 路由只写入 Windows `ActiveStore`（临时路由），系统重启后自然消失。
- 只管理本工具添加的路由：清理时精确匹配目的前缀、接口、网关与 metric，不覆盖、不删除外部路由。
- 启动前解析网卡网络环境：网卡不存在或没有 IPv4 网关时拒绝启动；IPv6 `/128` 路由仅在网卡同时具备全局 IPv6 地址与 IPv6 默认网关时启用。
- 配置不保存代理 UUID、密码、密钥、证书或订阅地址等敏感信息。

## 项目结构

```
├── main.go                 # Wails v3 应用入口(窗口/托盘/单实例/生命周期)
├── routes_service.go       # 路由规则 CRUD 绑定服务
├── nics_service.go         # 网卡枚举绑定服务
├── engine_service.go       # 核心服务启停/状态绑定服务
├── settings_service.go     # 全局设置读写绑定服务
├── internal/
│   ├── dns/                # DNS 解析器
│   ├── route/              # 主机路由注入与清理(Windows 实现)
│   ├── nics/               # 物理网卡枚举
│   ├── service/            # 路由服务引擎(轮询/对账/崩溃恢复)
│   └── store/              # JSON 本地持久化
└── frontend/
    ├── bindings/           # wails3 生成的 TS 绑定(wails3 generate bindings)
    ├── src/components/     # 界面组件(标题栏/控制栏/规则/设置/日志)
    ├── src/hooks/          # 领域状态管理
    └── src/api/            # 后端接口调用封装
```
