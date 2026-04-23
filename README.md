# OCI Ping CLI

一个用于测试 Oracle Cloud Infrastructure (OCI) 各个区域网络延迟的 Go 语言命令行工具。

## 功能特点

- **多平台支持**：支持 macOS (ARM), Windows (x64), Linux (x86_64) 和 Linux (ARM64)。
- **并发探测**：使用 Goroutines 并发测试所有区域的延迟，速度极快。
- **详细指标**：显示每个区域的最小、平均、最大和中位数延迟。
- **彩色输出**：在终端中使用颜色标识不同的延迟范围（仅限 Mac/Linux）。
- **进度显示**：实时显示探测进度。
- **结果导出**：自动将测试结果保存为带时间戳的 CSV 文件。
- **自定义配置**：支持通过 `-n` 指定每个区域的探测次数，或通过 `--regions-list` 使用自定义的区域 JSON。

## 安装与运行

### 使用脚本 (Mac/Linux)

项目根目录提供了一个 `oci-ping.sh` 脚本，可以自动检测您的系统并运行对应的二进制文件：

```bash
chmod +x oci-ping.sh
./oci-ping.sh -n 5
```

### 直接运行二进制文件

您可以直接运行对应平台的二进制文件：

- **Mac (ARM)**: `./oci-ping-cli-darwin-arm64`
- **Windows**: `oci-ping-cli-win-x64.exe` (建议以管理员权限运行以支持 ICMP)
- **Linux**: `./oci-ping-cli-linux-x64`

## 命令行参数

- `-n`: 每个区域的 ping 次数（默认为 10）。
- `--regions-list`: 指定区域 JSON 文件的路径或 URL。
- `-v`: 启用详细输出。

## 延迟颜色说明 (终端)

- **绿色**: < 100ms (延迟低)
- **黄色**: 100ms - 200ms (延迟中等)
- **橙色**: 200ms - 300ms (延迟较高)
- **红色**: > 300ms (延迟高)

## 数据来源

默认从 GitHub 获取最新的 OCI 区域列表：`https://ghfast.top/raw.githubusercontent.com/mark-floyd/oci-ping/refs/heads/main/regions.json`

## 许可证

MIT
