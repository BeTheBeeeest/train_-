package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
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
	CreateTime string    `json:"create_time"`
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

// 测试客户端
type TestClient struct {
	ID   int
	Conn net.Conn
}

// 创建测试客户端
func NewTestClient(id int, serverAddr string) (*TestClient, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return nil, err
	}

	return &TestClient{
		ID:   id,
		Conn: conn,
	}, nil
}

// 发送消息并接收响应
func (tc *TestClient) sendMessage(msgType string, data interface{}) (*Message, error) {
	var jsonData json.RawMessage
	if data != nil {
		bytes, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		jsonData = bytes
	}

	msg := Message{
		Type: msgType,
		Data: jsonData,
	}

	jsonMsg, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}

	_, err = tc.Conn.Write(append(jsonMsg, '\n'))
	if err != nil {
		return nil, err
	}

	// 读取响应
	buffer := make([]byte, 1024)
	n, err := tc.Conn.Read(buffer)
	if err != nil {
		return nil, err
	}

	var response Message
	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// 尝试购票
func (tc *TestClient) tryBuyTicket() {
	passenger := Passenger{
		ID:   fmt.Sprintf("ID_%d", tc.ID),
		Name: fmt.Sprintf("乘客_%d", tc.ID),
	}

	req := BuyRequest{
		Passenger: passenger,
		TrainID:   "G1001",
		Date:      "2024-01-01",
	}

	response, err := tc.sendMessage(MsgBuyTicket, req)
	if err != nil {
		log.Printf("客户端%d: 购票请求失败 - %v", tc.ID, err)
		return
	}

	if response.Success {
		fmt.Printf("✅ 客户端%d: %s\n", tc.ID, response.Message)
	} else {
		fmt.Printf("❌ 客户端%d: %s，尝试加入候补\n", tc.ID, response.Message)
		tc.addToWaitList()
	}
}

// 加入候补
func (tc *TestClient) addToWaitList() {
	passenger := Passenger{
		ID:   fmt.Sprintf("ID_%d", tc.ID),
		Name: fmt.Sprintf("乘客_%d", tc.ID),
	}

	req := WaitRequest{
		Passenger: passenger,
		TrainID:   "G1001",
		Date:      "2024-01-01",
	}

	response, err := tc.sendMessage(MsgWaitList, req)
	if err != nil {
		log.Printf("客户端%d: 候补请求失败 - %v", tc.ID, err)
		return
	}

	if response.Success {
		var order WaitOrder
		json.Unmarshal(response.Data, &order)
		fmt.Printf("🎫 客户端%d: 候补成功，订单号: %s，排队位置: 第%d位\n", 
			tc.ID, order.OrderID, order.Position)
	} else {
		fmt.Printf("❌ 客户端%d: %s\n", tc.ID, response.Message)
	}
}

// 关闭连接
func (tc *TestClient) Close() {
	tc.Conn.Close()
}

// 并发测试函数
func testConcurrentClients(clientCount int, serverAddr string) {
	var wg sync.WaitGroup
	
	fmt.Printf("🚀 开始并发测试，客户端数量: %d\n", clientCount)
	fmt.Printf("📍 服务器地址: %s\n", serverAddr)
	fmt.Println(strings.Repeat("=", 50))

	// 创建并启动多个客户端
	for i := 1; i <= clientCount; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()

			// 随机延迟，模拟真实场景
			time.Sleep(time.Duration(clientID*100) * time.Millisecond)

			client, err := NewTestClient(clientID, serverAddr)
			if err != nil {
				log.Printf("客户端%d: 连接失败 - %v", clientID, err)
				return
			}
			defer client.Close()

			fmt.Printf("🔗 客户端%d: 已连接到服务器\n", clientID)
			
			// 尝试购票
			client.tryBuyTicket()
		}(i)
	}

	// 等待所有客户端完成
	wg.Wait()
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("✅ 并发测试完成")
}

// 模拟退票测试
func testRefundScenario(serverAddr string) {
	fmt.Println("\n🔄 开始退票测试...")
	
	// 创建一个客户端进行退票
	client, err := NewTestClient(999, serverAddr)
	if err != nil {
		log.Printf("退票测试: 连接失败 - %v", err)
		return
	}
	defer client.Close()

	// 发送退票请求
	response, err := client.sendMessage(MsgRefund, nil)
	if err != nil {
		log.Printf("退票测试: 请求失败 - %v", err)
		return
	}

	if response.Success {
		fmt.Printf("✅ 退票成功: %s\n", response.Message)
		fmt.Println("💡 候补队列中的第一位乘客应该会自动获得车票")
	} else {
		fmt.Printf("❌ 退票失败: %s\n", response.Message)
	}
}

func main() {
	serverAddr := "localhost:8081"  // 改为8081端口
	
	fmt.Println("🎯 火车票候补系统并发测试")
	fmt.Println("请确保服务器已启动 (go run server/main.go)")
	fmt.Println()

	// 等待用户确认
	fmt.Print("按回车键开始测试...")
	fmt.Scanln()

	// 测试场景1: 10个客户端并发购票
	testConcurrentClients(10, serverAddr)

	// 等待一段时间
	time.Sleep(2 * time.Second)

	// 测试场景2: 模拟退票，触发候补队列
	testRefundScenario(serverAddr)

	// 测试场景3: 更多客户端加入候补
	fmt.Println("\n🔄 添加更多候补客户端...")
	time.Sleep(1 * time.Second)
	testConcurrentClients(5, serverAddr)

	fmt.Println("\n🎉 测试完成！")
	fmt.Println("💡 提示: 可以使用客户端程序查询候补位置和状态")
}
