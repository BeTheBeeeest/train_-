package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// 消息类型
const (
	MsgQueryTicket = "query_ticket"
	MsgBuyTicket   = "buy_ticket"
	MsgWaitList    = "wait_list"
	MsgRefund      = "refund"
	MsgCheckWait   = "check_wait"
	MsgResponse    = "response"
)

// 车次信息
type Train struct {
	ID    string `json:"id"`
	From  string `json:"from"`
	To    string `json:"to"`
	Total int    `json:"total"`
	Sold  int    `json:"sold"`
	Date  string `json:"date"`
}

// 乘客信息
type Passenger struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// 候补订单
type WaitOrder struct {
	OrderID    string    `json:"order_id"`
	Passenger  Passenger `json:"passenger"`
	TrainID    string    `json:"train_id"`
	Date       string    `json:"date"`
	CreateTime time.Time `json:"create_time"`
	Position   int       `json:"position"`
	Status     string    `json:"status"`
}

// 消息结构
type Message struct {
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data"`
	Success bool            `json:"success"`
	Message string          `json:"message"`
}

// 购票请求
type BuyRequest struct {
	Passenger Passenger `json:"passenger"`
	TrainID   string    `json:"train_id"`
	Date      string    `json:"date"`
}

// 候补请求
type WaitRequest struct {
	Passenger Passenger `json:"passenger"`
	TrainID   string    `json:"train_id"`
	Date      string    `json:"date"`
}

// 查询候补请求
type CheckWaitRequest struct {
	OrderID string `json:"order_id"`
}

// 票务系统
type TicketSystem struct {
	mu         sync.RWMutex
	Train      *Train
	WaitQueue  []*WaitOrder
	Orders     map[string]*WaitOrder
	OrderCount int
}

// 创建新的票务系统
func NewTicketSystem() *TicketSystem {
	train := &Train{
		ID:    "G1001",
		From:  "武汉",
		To:    "北京",
		Total: 100,
		Sold:  0,
		Date:  time.Now().Format("2006-01-02"),
	}

	return &TicketSystem{
		Train:      train,
		WaitQueue:  make([]*WaitOrder, 0),
		Orders:     make(map[string]*WaitOrder),
		OrderCount: 0,
	}
}

// 查询余票
func (ts *TicketSystem) QueryTicket() *Train {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.Train
}

// 购买车票
func (ts *TicketSystem) BuyTicket(passenger Passenger) (bool, string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.Train.Sold < ts.Train.Total {
		ts.Train.Sold++
		return true, "购票成功"
	}
	return false, "车票已售罄，可选择候补"
}

// 加入候补队列
func (ts *TicketSystem) AddToWaitList(req WaitRequest) *WaitOrder {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	ts.OrderCount++
	order := &WaitOrder{
		OrderID:    fmt.Sprintf("ORDER_%d", ts.OrderCount),
		Passenger:  req.Passenger,
		TrainID:    req.TrainID,
		Date:       req.Date,
		CreateTime: time.Now(),
		Position:   len(ts.WaitQueue) + 1,
		Status:     "waiting",
	}

	ts.WaitQueue = append(ts.WaitQueue, order)
	ts.Orders[order.OrderID] = order

	return order
}

// 查询候补位置
func (ts *TicketSystem) CheckWaitPosition(orderID string) (*WaitOrder, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	order, exists := ts.Orders[orderID]
	if !exists {
		return nil, false
	}

	// 更新排队位置
	for i, waitOrder := range ts.WaitQueue {
		if waitOrder.OrderID == orderID {
			order.Position = i + 1
			break
		}
	}

	return order, true
}

// 退票处理
func (ts *TicketSystem) ProcessRefund() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.Train.Sold > 0 {
		ts.Train.Sold--

		// 检查候补队列
		if len(ts.WaitQueue) > 0 {
			// 为候补队列第一个乘客购票
			firstOrder := ts.WaitQueue[0]
			firstOrder.Status = "success"

			// 从候补队列移除
			ts.WaitQueue = ts.WaitQueue[1:]

			// 更新队列中其他乘客的位置
			for i, order := range ts.WaitQueue {
				order.Position = i + 1
			}

			ts.Train.Sold++ // 候补购票成功，票数增加
		}
	}
}

// 客户端连接信息
type Client struct {
	Conn net.Conn
	ID   string
}

// 票务服务器
type TicketServer struct {
	System  *TicketSystem
	Clients map[string]*Client
	mu      sync.RWMutex
}

// 创建新服务器
func NewTicketServer() *TicketServer {
	return &TicketServer{
		System:  NewTicketSystem(),
		Clients: make(map[string]*Client),
	}
}

// 处理客户端连接
func (ts *TicketServer) HandleClient(conn net.Conn) {
	defer conn.Close()

	clientID := conn.RemoteAddr().String()
	client := &Client{
		Conn: conn,
		ID:   clientID,
	}

	ts.mu.Lock()
	ts.Clients[clientID] = client
	ts.mu.Unlock()

	log.Printf("客户端 %s 已连接", clientID)

	// 处理客户端消息
	buffer := make([]byte, 1024)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("客户端 %s 断开连接", clientID)
			ts.mu.Lock()
			delete(ts.Clients, clientID)
			ts.mu.Unlock()
			return
		}

		var msg Message
		if err := json.Unmarshal(buffer[:n], &msg); err != nil {
			ts.sendError(conn, "消息解析错误")
			continue
		}

		ts.handleMessage(conn, msg)
	}
}

// 处理消息
func (ts *TicketServer) handleMessage(conn net.Conn, msg Message) {
	switch msg.Type {
	case MsgQueryTicket:
		ts.handleQueryTicket(conn)
	case MsgBuyTicket:
		ts.handleBuyTicket(conn, msg.Data)
	case MsgWaitList:
		ts.handleWaitList(conn, msg.Data)
	case MsgCheckWait:
		ts.handleCheckWait(conn, msg.Data)
	case MsgRefund:
		ts.handleRefund(conn)
	default:
		ts.sendError(conn, "未知消息类型")
	}
}

// 处理查询余票
func (ts *TicketServer) handleQueryTicket(conn net.Conn) {
	train := ts.System.QueryTicket()
	ts.sendResponse(conn, true, "查询成功", train)
}

// 处理购票
func (ts *TicketServer) handleBuyTicket(conn net.Conn, data json.RawMessage) {
	var req BuyRequest
	if err := json.Unmarshal(data, &req); err != nil {
		ts.sendError(conn, "请求数据解析错误")
		return
	}

	success, message := ts.System.BuyTicket(req.Passenger)
	ts.sendResponse(conn, success, message, nil)
}

// 处理候补
func (ts *TicketServer) handleWaitList(conn net.Conn, data json.RawMessage) {
	var req WaitRequest
	if err := json.Unmarshal(data, &req); err != nil {
		ts.sendError(conn, "请求数据解析错误")
		return
	}

	order := ts.System.AddToWaitList(req)
	ts.sendResponse(conn, true, "候补成功", order)
}

// 处理查询候补
func (ts *TicketServer) handleCheckWait(conn net.Conn, data json.RawMessage) {
	var req CheckWaitRequest
	if err := json.Unmarshal(data, &req); err != nil {
		ts.sendError(conn, "请求数据解析错误")
		return
	}

	order, exists := ts.System.CheckWaitPosition(req.OrderID)
	if !exists {
		ts.sendResponse(conn, false, "订单不存在", nil)
		return
	}

	ts.sendResponse(conn, true, "查询成功", order)
}

// 处理退票
func (ts *TicketServer) handleRefund(conn net.Conn) {
	ts.System.ProcessRefund()
	ts.sendResponse(conn, true, "退票成功", nil)
}

// 发送响应
func (ts *TicketServer) sendResponse(conn net.Conn, success bool, message string, data interface{}) {
	response := Message{
		Type:    MsgResponse,
		Success: success,
		Message: message,
	}

	if data != nil {
		jsonData, _ := json.Marshal(data)
		response.Data = jsonData
	}

	jsonResponse, _ := json.Marshal(response)
	conn.Write(jsonResponse)
}

// 发送错误
func (ts *TicketServer) sendError(conn net.Conn, message string) {
	ts.sendResponse(conn, false, message, nil)
}

// 启动服务器
func (ts *TicketServer) Start(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal("服务器启动失败:", err)
	}
	defer listener.Close()

	log.Printf("票务服务器启动，监听端口 %s", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接错误: %v", err)
			continue
		}

		go ts.HandleClient(conn)
	}
}

func main() {
	port := "8081" // 改为8081端口，避免冲突
	log.Printf("启动票务服务器，端口: %s", port)
	server := NewTicketServer()
	server.Start(port)
}
