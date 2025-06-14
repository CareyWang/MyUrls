# sync_data - Redis 到 SQLite 数据同步工具

## 功能描述

sync_data 是一个用于将 Redis 数据一次性同步到 SQLite 的命令行工具，能完整保持原有的 TTL（过期时间）不变。

## 主要特性

- 🔄 **一次性同步**: 扫描所有 Redis keys 并一次性同步到 SQLite
- ⏰ **TTL 保持**: 完整保持 Redis 中设置的过期时间
- 📦 **批量处理**: 支持批量同步以提高性能
- 📊 **详细日志**: 提供详细的同步进度和结果日志

## 使用方法

### 编译

```bash
go build -o sync_data ./cmd/sync_data
```

### 命令行参数

```bash
./sync_data [options]

选项:
  -h                    显示帮助信息
  -redis-addr string    Redis 地址 (默认 "localhost:6379")
  -redis-password string Redis 密码 (默认 "")
  -sqlite-file string   SQLite 文件路径 (默认 "./data/sync_data.db")
  -batch-size int       批量处理大小 (默认 100)
```

### 环境变量配置

可以通过环境变量覆盖默认配置：

```bash
export SYNC_REDIS_ADDR="localhost:6379"
export SYNC_REDIS_PASSWORD="your_password"
export SYNC_SQLITE_FILE="./data/sync_data.db"
```

### 使用示例

1. **基本使用**：
```bash
./sync_data
```

2. **自定义配置**：
```bash
./sync_data -redis-addr="192.168.1.100:6379" -redis-password="secret" -batch-size=200
```

3. **使用环境变量**：
```bash
export SYNC_REDIS_ADDR="redis.example.com:6379"
export SYNC_REDIS_PASSWORD="your_password"
./sync_data
```

## 同步逻辑

1. **扫描 Redis**: 使用 SCAN 命令获取所有 keys
2. **读取数据**: 获取每个 key 的 value 和 TTL
3. **保持 TTL**: 
   - TTL = -1 (永不过期): 设置 100 年过期时间
   - TTL > 0: 保持原有过期时间
   - TTL = -2 (key 不存在): 跳过
4. **写入 SQLite**: 使用相同的 TTL 写入 SQLite

## 注意事项

- 确保 Redis 和 SQLite 的连接正常
- 同步过程中会输出详细日志
- 大量数据同步时请适当调整批量处理大小
- 确保有足够的磁盘空间存储 SQLite 数据
- 建议在低峰期进行大规模数据同步
- 程序执行完成后会自动退出

## 日志输出示例

```
2024/01/01 12:00:00 sync_data app 启动中...
2024/01/01 12:00:00 Redis 连接成功
2024/01/01 12:00:00 SQLite 连接成功
2024/01/01 12:00:00 开始一次性数据同步，批量大小: 100
2024/01/01 12:00:01 开始同步数据...
2024/01/01 12:00:01 找到 1500 个 Redis keys
2024/01/01 12:00:05 同步完成: 成功 1500 个，失败 0 个批次
2024/01/01 12:00:05 同步完成，程序退出
``` 