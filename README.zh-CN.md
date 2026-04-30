# BigText Reader

[English](README.md) | 简体中文

GitHub 仓库：[github.com/weiwentao996/bigtext-reader](https://github.com/weiwentao996/bigtext-reader)

![BigText Reader](build/appicon.png)

BigText Reader 是一个面向超大文本文件的桌面阅读器，适合打开 GB 级 `.txt`、`.log` 文件。它不会一次性把整个文件读入内存，而是按需分页读取，适合阅读大体积小说、资料文本、导出文件和日志文件。

## 关键词

大文本阅读器、超大文本阅读器、大文件阅读器、TXT 阅读器、LOG 查看器、GB 级文本查看器、日志查看器、大文本搜索、GBK 文本阅读器、UTF-8 文本阅读器、桌面文本阅读器

## 功能特性

- **流畅打开超大文件**：针对 GB 级 TXT / LOG 文件设计，避免普通编辑器打开大文件时卡死。
- **按字节偏移分页**：适合超大文件的快速定位和翻页。
- **连续滚动阅读**：滚动时自动加载上下文内容，阅读体验更自然。
- **支持 UTF-8 / GBK**：适合中文小说、日志、历史文本等不同编码场景。
- **全文搜索**：支持构建轻量搜索索引，统计并展示所有命中结果。
- **搜索结果虚拟滚动**：大量搜索结果也能无感滚动，不需要用户手动分页。
- **精确跳转搜索结果**：点击搜索结果可准确跳转到对应行和命中位置。
- **命中高亮**：阅读区和搜索预览中高亮关键词。
- **书签功能**：支持添加、跳转和删除书签。
- **阅读进度保存**：自动记录每个文件的阅读位置。
- **文件夹列表**：打开文件夹后可在侧边栏快速切换文件。
- **编码热切换**：打开文件后也可以切换编码并刷新内容。
- **字号调节**：适合长时间阅读。
- **中英文界面**：内置中文和英文切换。
- **桌面应用**：基于 Go + Wails 构建，轻量独立。

## 适用场景

- 阅读超大 TXT 小说、资料、导出文本。
- 查看普通编辑器打不开或打开很慢的 LOG 日志。
- 在 GB 级文本中搜索关键词。
- 打开 GBK 编码的中文文本。
- 记录长文本阅读进度和书签。
- 快速浏览一个文件夹内的大量日志或文本文件。

## 截图

### 大文件阅读

打开并滚动阅读 GB 级日志文件，同时显示阅读进度、编码和字号控制。

![大文件阅读](docs/screenshots/main-reader.png)

### 全文搜索

在整个文件中搜索关键词，浏览虚拟滚动结果，并跳转到高亮命中位置。

![全文搜索结果](docs/screenshots/main-search-results.png)

### 书签与搜索

保存阅读位置为书签，并结合搜索结果快速定位。

![书签与搜索](docs/screenshots/bookmarks-search.png)

## 支持的平台

| 平台 | 状态 | 说明 |
| --- | --- | --- |
| Windows 10 / 11 | 已支持 | 当前主要目标平台，发布产物为 `bigtext-reader.exe`。 |
| macOS | 可从源码构建 | 暂未提供发布包，也尚未完整测试。Wails 支持 macOS 构建。 |
| Linux | 可从源码构建 | 暂未提供发布包，也尚未完整测试。Wails 支持 Linux 构建，需要安装对应 WebKit 依赖。 |

BigText Reader 目前主要在 Windows 上开发和测试。macOS 与 Linux 发布包会作为后续计划完善。

## 安装

从 [GitHub Releases](https://github.com/weiwentao996/bigtext-reader/releases) 下载最新版并运行：

- Windows：`bigtext-reader.exe`

如果暂时没有发布包，可以按下面的步骤从源码构建。

## 从源码构建

### 环境要求

- Go 1.18+
- Node.js 和 npm
- Wails v2 CLI

安装 Wails：

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

开发模式：

```bash
wails dev
```

生产构建：

```bash
wails build
```

Windows 可执行文件会生成在：

```text
build/bin/bigtext-reader.exe
```

只构建前端：

```bash
cd frontend
npm install
npm run build
```

运行测试：

```bash
go test ./...
```

## 技术栈

- **Go**：后端文件读取、编码处理、搜索和状态持久化。
- **Wails v2**：桌面应用壳和 Go / JavaScript 通信。
- **Vanilla JavaScript**：无重型前端框架依赖。
- **Vite**：前端开发和构建工具。

## 项目结构

```text
bigtext-reader/
├── app.go                 # Wails 后端 API
├── main.go                # 应用入口
├── internal/
│   ├── reader/            # 大文件读取、分页、编码、搜索
│   └── state/             # 阅读进度和书签持久化
├── frontend/
│   ├── src/main.js        # UI 逻辑和国际化
│   └── src/style.css      # 应用样式
├── build/                 # 图标、manifest、构建产物
└── wails.json             # Wails 项目配置
```

## 数据与兼容性

BigText Reader 会在用户配置目录中保存本地阅读状态，例如阅读进度、编码和书签。

项目曾用名为 `bf-reader`。当前版本使用 `bigtext-reader`，并保留旧数据迁移兼容：

- 旧应用状态：`bf-reader` → `bigtext-reader`
- 旧前端偏好：`bf-reader.language` / `bf-reader.fontSize` → `bigtext-reader.language` / `bigtext-reader.fontSize`

这些兼容代码是有意保留的，用于避免旧用户升级后丢失阅读进度和设置。

## 计划

后续可考虑：

- 深色模式。
- 更多文本编码。
- 正则搜索。
- 大小写敏感 / 不敏感搜索选项。
- 导出搜索结果。
- Windows 安装包。
- macOS 和 Linux 发布包。

## 参与贡献

欢迎在 [GitHub 仓库](https://github.com/weiwentao996/bigtext-reader) 提交 Issue 和 Pull Request。

如果你发现某个大文本文件无法顺畅打开，建议提交 [Issue](https://github.com/weiwentao996/bigtext-reader/issues) 并说明：

- 操作系统。
- 文件大小。
- 文件编码，如果已知。
- 哪个操作慢或不正确。
- 文件类型是 TXT、LOG，还是其他纯文本格式。

## 许可证

暂未添加许可证。如果准备公开发布，建议添加 MIT、Apache-2.0 或 GPL-3.0 等开源许可证。

## 作者

作者：**weiwt**
