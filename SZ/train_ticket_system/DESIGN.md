# 火车票候补系统设计文档

## 1. 系统概述

本系统是一个基于Go语言开发的火车票候补系统，采用客户端-服务器(C/S)架构，支持多客户端并发访问。系统实现了完整的候补购票流程，包括查询、购票、候补、退票等功能。

## 2. 类模型设计

### 2.1 核心实体类

#### Train (车次信息)
```go
type Train struct {
    ID    string  // 车次号，如"G1001"
    From  string  // 出发地，如"武汉"
    To    string  // 目的地，如"北京"
    Total int     // 总票数，固定100张
    Sold  int     // 已售票数
    Date  string  // 发车日期
}
```
**职责**: 存储车次基本信息和票务状态

#### Passenger (乘客信息)
```go
type Passenger struct {
    ID   string  // 乘客身份证号
    Name string  // 乘客姓名
}
```
**职责**: 存储乘客基本信息

#### WaitOrder (候补订单)
```go
type WaitOrder struct {
    OrderID    string    // 订单号，格式"ORDER_序号"
    Passenger  Passenger // 乘客信息
    TrainID    string    // 车次号
    Date       string    // 乘车日期
    CreateTime time.Time // 创建时间
    Position   int       // 排队位置
    Status     string    // 状态: waiting/success/failed
}
```
**职责**: 存储候补订单信息和状态

### 2.2 业务逻辑类

#### TicketSystem (票务系统核心)
```go
type TicketSystem struct {
    mu         sync.RWMutex           // 读写锁，保证并发安全
    Train      *Train                 // 车次信息
    WaitQueue  []*WaitOrder          // 候补队列
    Orders     map[string]*WaitOrder  // 订单映射表
    OrderCount int                   // 订单计数器
}
```
**职责**: 
- 管理车票库存
- 维护候补队列
- 处理购票、候补、退票逻辑
- 保证并发安全

**核心方法**:
- `QueryTicket()`: 查询余票信息
- `BuyTicket(passenger)`: 购买车票
- `AddToWaitList(req)`: 加入候补队列
- `CheckWaitPosition(orderID)`: 查询候补位置
- `ProcessRefund()`: 处理退票和自动分配

#### TicketServer (网络服务器)
```go
type TicketServer struct {
    System  *TicketSystem           // 票务系统实例
    Clients map[string]*Client      // 客户端连接管理
    mu      sync.RWMutex           // 客户端连接锁
}
```
**职责**:
- 管理客户端连接
- 处理网络通信
- 消息路由和分发

#### TicketClient (客户端)
```go
type TicketClient struct {
    Conn net.Conn  // TCP连接
}
```
**职责**:
- 与服务器通信
- 用户界面交互
- 消息序列化和反序列化

### 2.3 消息传输类

#### Message (通信消息)
```go
type Message struct {
    Type    string          // 消息类型
    Data    json.RawMessage // 消息数据
    Success bool            // 操作结果
    Message string          // 响应消息
}
```
**职责**: 统一的客户端-服务器通信格式

## 3. 候补功能活动图

### 3.1 候补订单创建流程

```
[用户发起候补请求] 
    ↓
[验证乘客信息] 
    ↓
[检查车票状态] 
    ↓
[车票是否售罄?] ——否——→ [提示直接购票]
    ↓ 是
[生成候补订单]
    ↓
[加入候补队列]
    ↓
[分配排队位置]
    ↓
[返回订单信息]
    ↓
[用户确认支付] (模拟)
    ↓
[候补订单生效]
```

### 3.2 候补队列自动分配流程

```
[退票事件触发]
    ↓
[获取系统锁]
    ↓
[减少已售票数]
    ↓
[检查候补队列是否为空?] ——是——→ [结束流程]
    ↓ 否
[获取队列第一个订单]
    ↓
[为该订单自动购票]
    ↓
[更新订单状态为"success"]
    ↓
[从候补队列移除该订单]
    ↓
[更新队列中其他订单位置]
    ↓
[增加已售票数]
    ↓
[释放系统锁]
    ↓
[通知相关客户端] (可扩展)
```

### 3.3 候补位置查询流程

```
[用户输入订单号]
    ↓
[发送查询请求]
    ↓
[服务器验证订单号]
    ↓
[订单是否存在?] ——否——→ [返回"订单不存在"]
    ↓ 是
[获取订单信息]
    ↓
[计算当前排队位置]
    ↓
[返回订单状态和位置]
    ↓
[客户端显示结果]
```

## 4. 并发安全设计

### 4.1 锁机制
- **读写锁 (sync.RWMutex)**: 
  - 查询操作使用读锁，允许多个并发读取
  - 修改操作使用写锁，确保数据一致性

### 4.2 临界区保护
- **票务操作**: 购票、退票、候补加入等操作都在锁保护下进行
- **队列管理**: 候补队列的增删改操作都是原子性的
- **位置更新**: 排队位置的计算和更新保证一致性

### 4.3 并发场景处理
- **多客户端同时购票**: 通过锁机制确保票数正确
- **同时加入候补**: 保证排队位置的正确分配
- **退票触发分配**: 确保候补队列的正确处理

## 5. 网络通信协议

### 5.1 消息类型定义
```go
const (
    MsgQueryTicket = "query_ticket"    // 查询余票
    MsgBuyTicket   = "buy_ticket"      // 购买车票
    MsgWaitList    = "wait_list"       // 加入候补
    MsgRefund      = "refund"          // 退票
    MsgCheckWait   = "check_wait"      // 查询候补
    MsgResponse    = "response"        // 服务器响应
)
```

### 5.2 通信流程
1. **客户端连接**: TCP连接建立
2. **消息发送**: JSON序列化 + 换行符分隔
3. **服务器处理**: 消息解析 → 业务逻辑 → 响应生成
4. **响应返回**: JSON格式的统一响应
5. **连接管理**: 连接断开时清理资源

## 6. 扩展设计考虑

### 6.1 数据持久化
- 可添加数据库层存储订单和队列信息
- 支持系统重启后数据恢复

### 6.2 通知机制
- 可集成邮件/短信通知
- WebSocket推送实时状态更新

### 6.3 多车次支持
- 扩展Train结构支持多车次
- 独立的候补队列管理

### 6.4 超时处理
- 候补订单超时自动取消
- 定时任务清理过期订单

## 7. 性能优化建议

### 7.1 内存优化
- 使用对象池减少GC压力
- 合理设置队列容量上限

### 7.2 网络优化
- 连接池管理
- 消息批处理

### 7.3 并发优化
- 细粒度锁设计
- 无锁数据结构应用

这个设计确保了系统的可扩展性、并发安全性和用户体验的优化。
