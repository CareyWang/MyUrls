# 把短链接创建规则移出 HTTP handler

状态：已完成 · 优先级：中 · 建议在[短链接映射 module](short-url-mapping-module.md) 之后进行

## 背景

短链接创建的领域规则目前放在 `handlers.go` 里：

- `normalizeLongURL`、`isHTTPURL`：长链接规范化和校验，包括 Base64 兼容。
- `shortKeyPattern`：用户指定短键的校验。
- `LongToShortHandler`：错误分类，以及用全局 `proto`、`domain` 拼接短链接。

这些校验是修复创建缺陷时加的，为了让修复保持最小，暂时留在 handler 所在文件。当前规则如下：

- 长链接必须是带主机名的 `http` 或 `https` URL，先按明文判断，不符合再尝试 Base64 解码。
- 用户指定的短键只能包含字母、数字、`-` 和 `_`。
- 规则只在创建时生效，已有映射仍可解析。

## 问题

- **规则绑定在 HTTP 上。** 新增调用方（比如 CLI 或批量导入）要重写一遍，否则就会绕过校验。
- **短链接拼接依赖包级变量。** handler 直接读取全局 `proto`、`domain`，测试也要依赖这些全局值。
- **“短键不能和路由冲突”只是隐含规则。** 目前能防住 `/logo.png`，靠的是短键不允许带点。`public.go` 和 `main.go` 新增不带点的路由时，还是可能和短键冲突。
- **测试不便。** `handlers_test.go` 的 `TestLongToShortHandler` 按用例名称 switch 组装场景，每个领域规则都要经过路由才能验证。

## 目标

HTTP handler 只负责绑定参数和映射响应；规范化、校验、错误分类和短链接拼接都归入短链接创建 module。

## 范围

- 把长链接规范化、长链接和短键校验移入短链接创建 module。
- 用该 module 的错误类型表达校验失败和短键冲突，由 handler 统一映射成响应码。
- 让短链接拼接不再读取全局 `proto`、`domain`。
- 把保留路径变成显式规则，并与实际注册的路由保持一致。
- 把按名称 switch 的测试拆成领域用例，再保留少量 HTTP 映射用例。

## 不在范围内

- 改变现有响应结构，包括 `Code: 1` 的旧成功响应，以及 HTTP 200 加业务码的约定。
- 移除 Base64 兼容：内置前端 `public/index.html` 仍通过 `btoa()` 发送长链接。
- 校验或清理已有映射。
- 配置解析和启动流程的整体调整。

## 验收标准

- [x] `handlers.go` 不再直接做长链接解析、短键匹配或短链接字符串拼接。
- [x] 无需启动路由就能测试所有创建规则。
- [x] 以下输入都不会写入 Redis，并返回 `Code: 1001`：`example/path`、`example.com`、`javascript:alert(1)`、Base64 编码的 `javascript:alert(1)`、`https://`，以及短键 `logo.png`、`a/b`、`q?x`。
- [x] 明文、Base64、JSON 和表单四种提交方式继续可用，内置前端能正常创建短链接。
- [x] 与已注册路由冲突的短键会被拒绝，新增路由时有测试防止遗漏。
- [x] 响应码和 `Message` 文案不变，README 的 API 说明仍然准确。
- [x] `go test -race ./...` 通过。

## 已决问题

- **短键长度：** 上限 20 个字符。
- **保留路径：** 拒绝已注册的路由 `short`，并预留 `api`、`admin`。保留短键复用短键已被占用的 `Message`。
- **长链接长度：** 上限 8192 个字符，Base64 输入按解码后的长度计算。
- **自指链接：** 拒绝指向本服务域名的长链接。比较时忽略主机名大小写、末尾的点、协议差异和默认端口；非默认端口视为其他服务。

## 依赖

建议等[短链接映射 module](short-url-mapping-module.md) 完成后再做，这样创建规则可以直接放进收拢后的 module，避免移动两次。
