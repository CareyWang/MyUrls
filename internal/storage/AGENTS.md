## internal/storage

### OVERVIEW
抽象存储层，支持Redis和SQLite双驱动，内置线程安全LRU缓存。所有驱动实现Driver接口（interface.go），支持TTL过期机制。Manager（manager.go）负责驱动初始化和缓存访问。

### WHERE TO LOOK
- **接口定义**: interface.go - Driver接口定义9个核心方法
- **存储初始化**: manager.go - InitStorage工厂函数
- **Redis驱动**: redis.go - go-redis/v9封装，支持缓存层
- **SQLite驱动**: sqlite.go - GORM实现，后台goroutine清理过期数据（5分钟间隔）
- **LRU缓存**: lru.go - 基于container/list实现，后台goroutine清理过期项（1分钟间隔）
- **测试**: storage_test.go - 通用testStorageDriver函数；manager_test.go - 初始化逻辑测试

### CONVENTIONS
- **缓存策略**: 先读缓存，未命中读取底层存储并回写缓存（redis.go:49-69, sqlite.go:123-161）
- **TTL传播**: SetEx/Expire操作同步更新缓存过期时间（redis.go:72-80, sqlite.go:174-178）
- **默认配置**: 缓存size=128, TTL=300s（redis.go:16-20, sqlite.go:51-56）
- **测试隔离**: Redis测试检测SKIP_REDIS_TESTS环境变量和可用性（storage_test.go:24-34）
- **错误处理**: SQLite模拟Redis错误返回，"redis: nil"表示key不存在（sqlite.go:136, 146）

### ANTI-PATTERNS
- 不要直接实例化RedisDriver/SQLiteDriver，使用InitStorage
- 不要在service层判断driver类型，使用GetDriver返回的Driver接口
- 不要手动调用LRUCache.cleanupLoop，自动启动于构造函数
- 不要假设driver已初始化，先调用InitStorage
