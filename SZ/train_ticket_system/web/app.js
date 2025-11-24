const { createApp } = Vue;

createApp({
    data() {
        return {
            // WebSocket连接
            ws: null,
            isConnected: false,
            serverUrl: 'localhost:8081',  // 改为8081端口
            
            // 车次信息
            trainInfo: null,
            
            // 乘客信息
            passenger: {
                id: '',
                name: ''
            },
            
            // 候补订单查询
            checkOrderId: '',
            
            // 我的候补订单列表
            myWaitOrders: [],
            
            // 我的车票订单列表
            myTicketOrders: [],
            
            // 并发测试
            concurrentCount: 10,  // 默认10个并发客户端
            
            // 系统消息
            messages: [],
            
            // 确认对话框
            showConfirmDialog: false,
            confirmDialog: {
                title: '',
                message: '',
                action: null
            }
        };
    },
    
    computed: {
        // 计算剩余票数
        remainingTickets() {
            if (!this.trainInfo) return 0;
            return this.trainInfo.total - this.trainInfo.sold;
        }
    },
    
    mounted() {
        // 页面加载时自动连接服务器
        this.connectServer();
    },
    
    methods: {
        // 连接服务器
        connectServer() {
            try {
                // 注意：浏览器中的WebSocket无法直接连接到TCP服务器
                // 需要在Go服务器端添加WebSocket支持
                // 这里我们使用模拟的TCP连接方式
                this.addMessage('info', '正在连接服务器...');
                
                // 由于浏览器限制，我们需要使用HTTP API方式
                // 这里先模拟连接成功
                setTimeout(() => {
                    this.isConnected = true;
                    this.addMessage('success', '服务器连接成功');
                    this.queryTicket();
                }, 500);
                
            } catch (error) {
                this.addMessage('error', '连接服务器失败: ' + error.message);
                this.isConnected = false;
            }
        },
        
        // 发送消息到服务器（模拟TCP通信）
        async sendMessage(type, data) {
            if (!this.isConnected) {
                this.addMessage('error', '未连接到服务器');
                return null;
            }
            
            const message = {
                type: type,
                data: data ? JSON.stringify(data) : null
            };
            
            // 这里需要通过HTTP代理或WebSocket网关与Go服务器通信
            // 为了演示，我们模拟服务器响应
            return this.simulateServerResponse(type, data);
        },
        
        // 模拟服务器响应（实际项目中应该通过HTTP API或WebSocket）
        simulateServerResponse(type, data) {
            return new Promise((resolve) => {
                setTimeout(() => {
                    let response = { success: false, message: '', data: null };
                    
                    switch (type) {
                        case 'query_ticket':
                            // 如果trainInfo已存在，保持已售票数不变
                            // 否则初始化为0
                            const currentSold = this.trainInfo ? this.trainInfo.sold : 0;
                            response = {
                                success: true,
                                message: '查询成功',
                                data: {
                                    id: 'G1001',
                                    from: '武汉',
                                    to: '北京',
                                    total: 100,
                                    sold: currentSold,
                                    date: new Date().toISOString().split('T')[0]
                                }
                            };
                            break;
                            
                        case 'buy_ticket':
                            const hasTickets = this.trainInfo && this.remainingTickets > 0;
                            if (hasTickets) {
                                // 生成车票订单
                                const ticketOrderId = 'TICKET_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
                                const ticketOrder = {
                                    order_id: ticketOrderId,
                                    passenger: data.passenger,
                                    train_id: data.train_id,
                                    date: data.date,
                                    purchase_time: new Date().toISOString(),
                                    status: 'confirmed'
                                };
                                
                                response = {
                                    success: true,
                                    message: '购票成功！',
                                    data: ticketOrder
                                };
                                if (this.trainInfo) {
                                    this.trainInfo.sold++;
                                }
                            } else {
                                response = {
                                    success: false,
                                    message: '车票已售罄，可选择候补'
                                };
                            }
                            break;
                            
                        case 'wait_list':
                            const orderId = 'ORDER_' + Date.now();
                            const order = {
                                order_id: orderId,
                                passenger: data.passenger,
                                train_id: data.train_id,
                                date: data.date,
                                create_time: new Date().toISOString(),
                                position: this.myWaitOrders.length + 1,
                                status: 'waiting'
                            };
                            response = {
                                success: true,
                                message: '候补成功',
                                data: order
                            };
                            break;
                            
                        case 'check_wait':
                            const foundOrder = this.myWaitOrders.find(o => o.order_id === data.order_id);
                            if (foundOrder) {
                                response = {
                                    success: true,
                                    message: '查询成功',
                                    data: foundOrder
                                };
                            } else {
                                response = {
                                    success: false,
                                    message: '订单不存在'
                                };
                            }
                            break;
                            
                        case 'refund':
                            if (this.trainInfo && this.trainInfo.sold > 0) {
                                this.trainInfo.sold--;
                                // 如果有候补订单，自动为第一位购票
                                if (this.myWaitOrders.length > 0) {
                                    const firstOrder = this.myWaitOrders[0];
                                    firstOrder.status = 'success';
                                    this.trainInfo.sold++;
                                    // 更新其他订单位置
                                    this.myWaitOrders.forEach((order, index) => {
                                        if (index > 0) {
                                            order.position = index;
                                        }
                                    });
                                }
                                response = {
                                    success: true,
                                    message: '退票成功'
                                };
                            } else {
                                response = {
                                    success: false,
                                    message: '没有可退的票'
                                };
                            }
                            break;
                    }
                    
                    resolve(response);
                }, 300);
            });
        },
        
        // 查询余票
        async queryTicket() {
            const response = await this.sendMessage('query_ticket', null);
            if (response && response.success) {
                this.trainInfo = response.data;
                this.addMessage('info', '余票查询成功');
            } else {
                this.addMessage('error', '查询失败: ' + (response?.message || '未知错误'));
            }
        },
        
        // 测试功能：将余票设置为0（售罄）
        sellOutTickets() {
            if (!this.trainInfo) {
                this.addMessage('warning', '请先查询车次信息');
                return;
            }
            
            this.showConfirm(
                '测试售罄',
                '确认将余票设置为0吗？\n\n这将模拟车票售罄的情况，方便测试候补功能。',
                () => {
                    // 将已售票数设置为总票数
                    this.trainInfo.sold = this.trainInfo.total;
                    this.addMessage('warning', '已将余票设置为0，现在只能加入候补队列');
                    this.addMessage('info', '提示：您可以点击"重置系统"重新开始');
                }
            );
        },
        
        // 重置系统
        resetSystem() {
            this.showConfirm(
                '重置系统',
                '确认重置系统吗？\n\n这将清空所有数据，包括：\n- 余票恢复为100张\n- 清空车票订单\n- 清空候补订单\n- 清空系统消息',
                () => {
                    // 重置车次信息
                    if (this.trainInfo) {
                        this.trainInfo.sold = 0;
                    }
                    // 清空车票订单
                    this.myTicketOrders = [];
                    // 清空候补订单
                    this.myWaitOrders = [];
                    // 清空系统消息
                    this.messages = [];
                    
                    this.addMessage('success', '✅ 系统已重置！余票: 100/100');
                    this.addMessage('info', '💡 现在可以重新开始测试');
                }
            );
        },
        
        // 购买车票
        async buyTicket() {
            if (!this.validatePassenger()) return;
            
            this.showConfirm(
                '确认购票',
                `确认为 ${this.passenger.name} 购买车票吗？`,
                async () => {
                    const request = {
                        passenger: { ...this.passenger },
                        train_id: this.trainInfo.id,
                        date: this.trainInfo.date
                    };
                    
                    const response = await this.sendMessage('buy_ticket', request);
                    if (response && response.success) {
                        // 保存车票订单
                        if (response.data) {
                            this.myTicketOrders.push(response.data);
                        }
                        
                        // 显示订单信息
                        const orderInfo = response.data;
                        this.addMessage('success', `✅ ${response.message}`);
                        this.addMessage('info', `🎫 车票订单号: ${orderInfo.order_id}`);
                        this.addMessage('info', `👤 乘客: ${orderInfo.passenger.name} (${orderInfo.passenger.id})`);
                        this.addMessage('info', `🚄 车次: ${orderInfo.train_id} | 日期: ${orderInfo.date}`);
                        
                        this.queryTicket();
                    } else {
                        this.addMessage('warning', response?.message || '购票失败');
                        // 询问是否加入候补
                        this.showConfirm(
                            '加入候补',
                            '车票已售罄，是否加入候补队列？',
                            () => this.addToWaitList()
                        );
                    }
                }
            );
        },
        
        // 加入候补队列
        async addToWaitList() {
            if (!this.validatePassenger()) return;
            
            this.showConfirm(
                '确认候补',
                `确认为 ${this.passenger.name} 加入候补队列吗？\n\n系统将在有票时自动为您购票。`,
                async () => {
                    const request = {
                        passenger: { ...this.passenger },
                        train_id: this.trainInfo.id,
                        date: this.trainInfo.date
                    };
                    
                    const response = await this.sendMessage('wait_list', request);
                    if (response && response.success) {
                        this.myWaitOrders.push(response.data);
                        this.addMessage('success', 
                            `候补成功！订单号: ${response.data.order_id}，排队位置: 第${response.data.position}位`
                        );
                    } else {
                        this.addMessage('error', '加入候补失败: ' + (response?.message || '未知错误'));
                    }
                }
            );
        },
        
        // 查询订单（支持车票和候补订单）
        async checkWaitPosition() {
            const orderId = this.checkOrderId.trim();
            
            if (!orderId) {
                this.addMessage('warning', '请输入订单号');
                return;
            }
            
            // 判断订单类型
            if (orderId.startsWith('TICKET_')) {
                // 查询车票订单
                this.checkTicketOrder(orderId);
            } else if (orderId.startsWith('ORDER_')) {
                // 查询候补订单
                this.checkWaitOrder(orderId);
            } else {
                this.addMessage('warning', '订单号格式不正确，应以 TICKET_ 或 ORDER_ 开头');
            }
        },
        
        // 查询车票订单
        checkTicketOrder(orderId) {
            const ticket = this.myTicketOrders.find(t => t.order_id === orderId);
            
            if (ticket) {
                this.addMessage('success', `✅ 找到车票订单: ${orderId}`);
                this.addMessage('info', `👤 乘客: ${ticket.passenger.name} (${ticket.passenger.id})`);
                this.addMessage('info', `🚄 车次: ${ticket.train_id} | 日期: ${ticket.date}`);
                this.addMessage('info', `⏰ 购票时间: ${this.formatTime(ticket.purchase_time)}`);
                this.addMessage('info', `📋 状态: 已出票`);
            } else {
                this.addMessage('error', `未找到车票订单: ${orderId}`);
                this.addMessage('info', '提示: 请检查订单号是否正确，或在"我的车票"区域查看');
            }
        },
        
        // 查询候补订单
        async checkWaitOrder(orderId) {
            const request = {
                order_id: orderId
            };
            
            const response = await this.sendMessage('check_wait', request);
            if (response && response.success) {
                const order = response.data;
                this.addMessage('success', `✅ 找到候补订单: ${order.order_id}`);
                this.addMessage('info', `👤 乘客: ${order.passenger.name} (${order.passenger.id})`);
                this.addMessage('info', `🚄 车次: ${order.train_id} | 日期: ${order.date}`);
                this.addMessage('info', `📊 状态: ${this.getStatusText(order.status)}`);
                this.addMessage('info', `🔢 排队位置: 第${order.position}位`);
                
                // 更新本地订单信息
                const index = this.myWaitOrders.findIndex(o => o.order_id === order.order_id);
                if (index !== -1) {
                    this.myWaitOrders[index] = order;
                } else {
                    this.myWaitOrders.push(order);
                }
            } else {
                this.addMessage('error', `未找到候补订单: ${orderId}`);
                this.addMessage('info', '提示: 请检查订单号是否正确，或在"我的候补订单"区域查看');
            }
        },
        
        // 退票
        async refundTicket() {
            this.showConfirm(
                '确认退票',
                '确认退票吗？如有候补订单，将自动为候补队列第一位乘客购票。',
                async () => {
                    const response = await this.sendMessage('refund', null);
                    if (response && response.success) {
                        this.addMessage('success', response.message);
                        this.queryTicket();
                        
                        // 检查是否有候补订单被兑现
                        if (this.myWaitOrders.length > 0 && this.myWaitOrders[0].status === 'success') {
                            this.addMessage('success', 
                                `候补订单 ${this.myWaitOrders[0].order_id} 已自动购票成功！`
                            );
                        }
                    } else {
                        this.addMessage('error', '退票失败: ' + (response?.message || '未知错误'));
                    }
                }
            );
        },
        
        // 验证乘客信息
        validatePassenger() {
            if (!this.passenger.id || !this.passenger.name) {
                this.addMessage('warning', '请填写完整的乘客信息');
                return false;
            }
            return true;
        },
        
        // 并发抢票测试
        async startConcurrentBuying() {
            if (!this.trainInfo) {
                this.addMessage('warning', '请先查询车次信息');
                return;
            }
            
            const count = this.concurrentCount;
            this.addMessage('info', `🚀 开始模拟 ${count} 个客户端并发抢票...`);
            
            let successCount = 0;  // 成功购票数
            let waitCount = 0;     // 加入候补数
            
            // 创建所有购票请求（模拟并发）
            const promises = [];
            for (let i = 1; i <= count; i++) {
                const clientId = `客户端${i}`;
                const passenger = {
                    id: `ID_${Date.now()}_${i}`,
                    name: `乘客${i}`
                };
                
                // 模拟并发购票
                const promise = this.simulateConcurrentBuy(clientId, passenger);
                promises.push(promise);
            }
            
            // 等待所有请求完成
            const results = await Promise.all(promises);
            
            // 统计结果
            results.forEach(result => {
                if (result.success) {
                    successCount++;
                } else if (result.waitlist) {
                    waitCount++;
                }
            });
            
            // 显示统计结果
            this.addMessage('success', `✅ 并发抢票完成！`);
            this.addMessage('info', `📊 成功购票: ${successCount} 张`);
            if (waitCount > 0) {
                this.addMessage('warning', `⏰ 加入候补: ${waitCount} 人`);
            }
            this.addMessage('info', `🎫 当前余票: ${this.remainingTickets} / ${this.trainInfo.total}`);
            
            // 刷新显示
            await this.queryTicket();
        },
        
        // 模拟单个客户端购票
        async simulateConcurrentBuy(clientId, passenger) {
            // 检查是否有余票
            if (this.trainInfo && this.remainingTickets > 0) {
                // 有票，购买成功
                this.trainInfo.sold++;
                return { 
                    success: true, 
                    clientId: clientId,
                    passenger: passenger
                };
            } else {
                // 无票，加入候补
                const orderId = 'ORDER_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
                const order = {
                    order_id: orderId,
                    passenger: passenger,
                    train_id: this.trainInfo.id,
                    date: this.trainInfo.date,
                    create_time: new Date().toISOString(),
                    position: this.myWaitOrders.length + 1,
                    status: 'waiting'
                };
                this.myWaitOrders.push(order);
                
                return { 
                    success: false, 
                    waitlist: true,
                    clientId: clientId,
                    order: order
                };
            }
        },
        
        // 添加消息
        addMessage(type, text) {
            const message = {
                type: type,
                text: text,
                time: new Date().toLocaleTimeString()
            };
            this.messages.unshift(message);
            
            // 最多保留20条消息
            if (this.messages.length > 20) {
                this.messages.pop();
            }
        },
        
        // 显示确认对话框
        showConfirm(title, message, action) {
            this.confirmDialog = {
                title: title,
                message: message,
                action: action
            };
            this.showConfirmDialog = true;
        },
        
        // 关闭确认对话框
        closeConfirmDialog() {
            this.showConfirmDialog = false;
        },
        
        // 确认操作
        confirmAction() {
            if (this.confirmDialog.action) {
                this.confirmDialog.action();
            }
            this.closeConfirmDialog();
        },
        
        // 获取状态文本
        getStatusText(status) {
            const statusMap = {
                'waiting': '等待中',
                'success': '已兑现',
                'failed': '已失败'
            };
            return statusMap[status] || status;
        },
        
        // 获取消息图标
        getMessageIcon(type) {
            const iconMap = {
                'success': 'fa-check-circle',
                'error': 'fa-exclamation-circle',
                'warning': 'fa-exclamation-triangle',
                'info': 'fa-info-circle'
            };
            return iconMap[type] || 'fa-info-circle';
        },
        
        // 格式化时间
        formatTime(timeString) {
            const date = new Date(timeString);
            return date.toLocaleString('zh-CN');
        }
    }
}).mount('#app');
