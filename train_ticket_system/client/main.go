package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
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

// 查询候补请求
type CheckWaitRequest struct {
	OrderID string `json:"order_id"`
}

// 票务客户端
type TicketClient struct {
	Conn net.Conn
}

// 创建新客户端
func NewTicketClient(serverAddr string) (*TicketClient, error) {
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		return nil, err
	}

	return &TicketClient{Conn: conn}, nil
}

// 发送消息并接收响应
func (tc *TicketClient) sendMessage(msgType string, data interface{}) (*Message, error) {
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

// 查询余票
func (tc *TicketClient) QueryTicket() {
	response, err := tc.sendMessage(MsgQueryTicket, nil)
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}

	if response.Success {
		var train Train
		json.Unmarshal(response.Data, &train)
		fmt.Printf("=== 车次信息 ===\n")
		fmt.Printf("车次: %s\n", train.ID)
		fmt.Printf("路线: %s → %s\n", train.From, train.To)
		fmt.Printf("日期: %s\n", train.Date)
		fmt.Printf("余票: %d/%d\n", train.Total-train.Sold, train.Total)
	} else {
		fmt.Printf("查询失败: %s\n", response.Message)
	}
}

// 购买车票
func (tc *TicketClient) BuyTicket(passenger Passenger) {
	req := BuyRequest{
		Passenger: passenger,
		TrainID:   "G1001",
		Date:      "2024-01-01",
	}

	response, err := tc.sendMessage(MsgBuyTicket, req)
	if err != nil {
		log.Printf("购票失败: %v", err)
		return
	}

	if response.Success {
		fmt.Printf("🎫 %s\n", response.Message)
	} else {
		fmt.Printf("❌ %s\n", response.Message)
	}
}

// 加入候补
func (tc *TicketClient) AddToWaitList(passenger Passenger) {
	req := WaitRequest{
		Passenger: passenger,
		TrainID:   "G1001",
		Date:      "2024-01-01",
	}

	response, err := tc.sendMessage(MsgWaitList, req)
	if err != nil {
		log.Printf("候补失败: %v", err)
		return
	}

	if response.Success {
		var order WaitOrder
		json.Unmarshal(response.Data, &order)
		fmt.Printf("✅ 候补成功!\n")
		fmt.Printf("订单号: %s\n", order.OrderID)
		fmt.Printf("排队位置: 第%d位\n", order.Position)
		fmt.Printf("请记下您的订单号以便查询: %s\n", order.OrderID)
	} else {
		fmt.Printf("❌ %s\n", response.Message)
	}
}

// 查询候补位置
func (tc *TicketClient) CheckWaitPosition(orderID string) {
	req := CheckWaitRequest{OrderID: orderID}

	response, err := tc.sendMessage(MsgCheckWait, req)
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}

	if response.Success {
		var order WaitOrder
		json.Unmarshal(response.Data, &order)
		fmt.Printf("=== 候补信息 ===\n")
		fmt.Printf("订单号: %s\n", order.OrderID)
		fmt.Printf("乘客: %s\n", order.Passenger.Name)
		fmt.Printf("排队位置: 第%d位\n", order.Position)
		fmt.Printf("状态: %s\n", order.Status)
	} else {
		fmt.Printf("❌ %s\n", response.Message)
	}
}

// 退票
func (tc *TicketClient) RefundTicket() {
	response, err := tc.sendMessage(MsgRefund, nil)
	if err != nil {
		log.Printf("退票失败: %v", err)
		return
	}

	if response.Success {
		fmt.Printf("✅ %s\n", response.Message)
		fmt.Printf("如果有候补乘客，系统会自动为候补队列第一位乘客购票\n")
	} else {
		fmt.Printf("❌ %s\n", response.Message)
	}
}

// 显示菜单
func (tc *TicketClient) ShowMenu() {
	fmt.Println("\n=== 火车票售票系统 ===")
	fmt.Println("1. 查询余票")
	fmt.Println("2. 购买车票")
	fmt.Println("3. 加入候补")
	fmt.Println("4. 查询候补位置")
	fmt.Println("5. 退票")
	fmt.Println("0. 退出")
	fmt.Print("请选择操作: ")
}

// 运行客户端
func (tc *TicketClient) Run() {
	defer tc.Conn.Close()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("欢迎使用火车票售票系统!")
	fmt.Println("车次: G1001 (武汉 → 北京)")

	for {
		tc.ShowMenu()

		if !scanner.Scan() {
			break
		}

		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			tc.QueryTicket()
		case "2":
			fmt.Print("请输入乘客姓名: ")
			scanner.Scan()
			name := strings.TrimSpace(scanner.Text())
			fmt.Print("请输入乘客ID: ")
			scanner.Scan()
			id := strings.TrimSpace(scanner.Text())

			passenger := Passenger{ID: id, Name: name}
			tc.BuyTicket(passenger)
		case "3":
			fmt.Print("请输入乘客姓名: ")
			scanner.Scan()
			name := strings.TrimSpace(scanner.Text())
			fmt.Print("请输入乘客ID: ")
			scanner.Scan()
			id := strings.TrimSpace(scanner.Text())

			passenger := Passenger{ID: id, Name: name}
			tc.AddToWaitList(passenger)
		case "4":
			fmt.Print("请输入订单号: ")
			scanner.Scan()
			orderID := strings.TrimSpace(scanner.Text())
			tc.CheckWaitPosition(orderID)
		case "5":
			tc.RefundTicket()
		case "0":
			fmt.Println("谢谢使用，再见!")
			return
		default:
			fmt.Println("无效选择，请重新输入")
		}

		fmt.Print("\n按回车键继续...")
		scanner.Scan()
	}
}

func main() {
	serverAddr := "localhost:8080"

	client, err := NewTicketClient(serverAddr)
	if err != nil {
		log.Fatalf("连接服务器失败: %v", err)
	}

	fmt.Println("成功连接到票务服务器")
	client.Run()
}
