# apitest

`apitest` 是一个基于 YAML 测试计划的命令行 HTTP 测试工具，支持变量模板、断言、提取上下文变量并输出 Markdown 报告。

## 快速开始

1. 安装依赖并构建：

```bash
cd apitest
go build ./cmd/apitest
```

2. 准备测试计划（参考 `examples/plan.yaml`）。

3. 运行：

```bash
./apitest run -f examples/plan.yaml -o report.md --var username=bob --base-url https://api.example.com
```

退出码约定：
- `0`：全部通过
- `1`：请求或断言失败
- `2`：配置或解析错误

## CLI 选项

| 选项 | 说明 |
| --- | --- |
| `-f, --file` | 指定 YAML 计划文件（必需） |
| `-o, --output` | Markdown 报告输出路径，默认 `report.md` |
| `--base-url` | 覆盖计划中的 `base_url` |
| `--var key=value` | 追加或覆盖变量，支持重复多次 |
| `--env env.yaml` | 加载额外变量（YAML map） |
| `--insecure` | 跳过 TLS 校验 |
| `--verbose` | 打印执行日志到 stdout |

## YAML 格式概要

```yaml
name: Demo Plan
base_url: https://example.com
vars:
  username: alice
  password: 123456
steps:
  - name: login
    request:
      method: POST
      url: /api/login
      headers:
        Content-Type: application/json
      body:
        json:
          username: "{{username}}"
          password: "{{password}}"
    extract:
      token:
        from: json
        path: data.token
    assert:
      - type: status
        op: ==
        expect: 200
```

- 模板：`{{var}}` 会被上下文变量替换，缺失变量会导致失败并终止。
- `body` 支持 `raw`、`json`、`form`，三者互斥。
- 断言类型：`status`、`header`、`body`、`json`（包含 `== != >= <= contains exists gt lt regex` 等操作符）。
- `extract` 支持从 `json`、`header`、`regex` 提取变量供后续步骤使用。

## 报告

运行后会生成 Markdown 报告，包含：
- 总览（起止时间、耗时、结果、失败步骤）
- 每个步骤的请求/响应详情、断言结果、提取变量
- 自动脱敏 `Authorization` 及键名含 `token/password/secret` 的值；响应体超过阈值会截断显示。

## 开发与测试

运行单元与集成测试：

```bash
go test ./...
```

代码组织：
- `internal/config`：YAML 结构与加载
- `internal/templ`：模板替换
- `internal/httpx`：请求构建与执行
- `internal/assert`：断言引擎
- `internal/runner`：执行器与上下文
- `internal/report`：Markdown 报告
- `cmd/apitest`：CLI 入口

## 示例

`examples/plan.yaml` 给出了一个包含登录与获取用户信息的示例计划，可直接修改 `base_url` 后运行。
