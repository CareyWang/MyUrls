# 把短链接映射收拢成一个 deep module

状态：已完成 · 优先级：高 · 建议先于[短链接创建规则](short-url-creation-rules.md)进行

## 背景

短键到长链接的映射知识分散在多个 module 里：

| 位置 | 承担的知识 |
| :--- | :--- |
| `creation.go` 的 `newShortURLCreator`、`Create` | 写入方式、有效期、不重发 SET NX 的重试策略 |
| `logic.go` 的 `ShortToLong`、`Renew` | 读取方式、错误转换、续期 |
| `redis.go` | 全局 `RedisClient` 及其 getter |
| `main.go` 的 `main`、`run` | 建立和探活客户端，再派生创建用客户端 |

## 问题

- **创建用客户端靠派生再撤销状态得到。** `newShortURLCreator` 复制主客户端已初始化的配置，再撤销其中的 `MaxRetries` 和 `PushNotificationProcessor`。升级 go-redis 时，漏撤销 push 处理器造成了启动报错，以后新增的初始化状态也可能重复这个问题。
- **读取依赖全局状态。** `ShortToLong` 通过 `GetRedisClient()` 取客户端，测试必须替换全局 `RedisClient`（见 `logic_test.go`、`handlers_test.go`、`main_test.go`）。
- **存在没有生产调用方的代码。** `Renew` 只被注释掉的代码引用，`handlers.go` 的 `defaultRenewTime` 同样没有生效。

## 目标

由一个 module 拥有短链接创建和解析，调用方不需要知道 Redis 客户端、重试策略或存储格式。

## 范围

- 把短链接创建、解析以及两种重试策略收进同一个 module。
- 从原始配置分别建立客户端，不再从已初始化的客户端派生。
- 移除全局 `RedisClient`，由 `main` 和 handler 持有这个 module。
- 决定 `Renew` 和 `defaultRenewTime` 的去留（见待决问题）。

## 不在范围内

- 短键和长链接的校验规则，见[短链接创建规则](short-url-creation-rules.md)。
- 引入 Redis interface 或 mock：目前只有一个生产 adapter，miniredis 已能覆盖测试需求。（SQLite 加入后这个前提不再成立，接口的引入见[存储后端与解析缓存](storage-sqlite-cache.md)。）
- 改变 HTTP 响应格式或现有映射的存储格式。

## 验收标准

- [x] 代码库不再有全局 `RedisClient` 和 `GetRedisClient`，测试也无需替换全局状态。
- [x] 创建用客户端不再复制其他客户端的已初始化配置，`newShortURLCreator` 里的撤销逻辑随之删除。
- [x] 丢失 SET 回复时仍只发送一次 SET，且返回存储错误。
- [x] 解析在“映射不存在”和“存储故障”时结果可区分，GET 分别返回 404 和 500。
- [x] 启动日志没有 `failed to initialize maintnotifications`。
- [x] 创建和解析的行为都通过这个 module 的 interface 用 miniredis 测试，`go test -race ./...` 通过。
- [x] 不再需要的测试被替换，而不是在新测试之外继续保留，例如对客户端配置细节的断言。

## 已决问题

- `Renew` 已删除，`defaultRenewTime` 一并移除。README 写明有效期固定为 365 天，需要访问时续期时再加。

## 风险

- 解析和创建的重试策略不同，这是有意的设计：重发 SET NX 可能把自己的写入变成短键冲突，这一点要保留在实现内部并由测试锁定。
- `main` 启动时的 Redis 探活需要继续生效，端口被占时也仍要以非零退出码退出。
