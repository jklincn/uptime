<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
  servers: {
    type: Array,
    required: true
  }
})

// 存储每个服务器的状态：{ power: 'on'|'off'|'checking'|'unknown', ssh: 'online'|'offline'|'checking'|'waiting'|'skipped' }
const serverStates = ref({})

// 确认对话框状态
const showConfirmModal = ref(false)
const pendingAction = ref(null) // { server: object, actionType: 'on'|'off' }

const checkStatus = async (server) => {
  // 初始化状态
  serverStates.value[server.id] = { power: 'checking', ssh: 'waiting' }
  
  // 1. 模拟检查电源状态 (BMC)
  await new Promise(resolve => setTimeout(resolve, 600 + Math.random() * 600))
  
  // 模拟 80% 概率开机
  const isPowerOn = Math.random() > 0.2
  
  // 更新电源状态
  serverStates.value[server.id].power = isPowerOn ? 'on' : 'off'
  
  // 2. 如果开机，继续检查 SSH
  if (isPowerOn) {
    serverStates.value[server.id].ssh = 'checking'
    
    await new Promise(resolve => setTimeout(resolve, 600 + Math.random() * 1000))
    
    // 模拟 90% SSH 可连接
    const isSshOnline = Math.random() > 0.1
    serverStates.value[server.id].ssh = isSshOnline ? 'online' : 'offline'
  } else {
    serverStates.value[server.id].ssh = 'skipped'
  }
}

const togglePower = (server) => {
  const currentState = serverStates.value[server.id]?.power
  if (!currentState || currentState === 'checking') return

  const actionType = currentState === 'on' ? 'off' : 'on'
  pendingAction.value = { server, actionType }
  showConfirmModal.value = true
}

const confirmAction = async () => {
  if (!pendingAction.value) return
  
  const { server } = pendingAction.value
  showConfirmModal.value = false
  pendingAction.value = null

  // 设置为检测中，模拟操作延迟
  serverStates.value[server.id].power = 'checking'
  serverStates.value[server.id].ssh = 'waiting'
  
  await new Promise(resolve => setTimeout(resolve, 1500))
  
  // 操作完成后重新检查状态
  checkStatus(server)
}

const cancelAction = () => {
  showConfirmModal.value = false
  pendingAction.value = null
}

const checkAll = () => {
  props.servers.forEach(server => {
    checkStatus(server)
  })
}

onMounted(() => {
  checkAll()
})
</script>

<template>
  <div class="monitor-container">
    <div class="header">
      <button @click="checkAll" class="refresh-btn">刷新列表状态</button>
    </div>
    
    <div class="server-list">
      <div class="list-header">
        <div class="col">服务器名称</div>
        <div class="col">IP 地址</div>
        <div class="col">电源状态</div>
        <div class="col">SSH 连接</div>
        <div class="col action-col">操作</div>
      </div>
      <div v-for="server in servers" :key="server.id" class="list-item">
        <div class="col name">{{ server.name }}</div>
        <div class="col ip">{{ server.ip }}</div>
        
        <!-- 电源状态 -->
        <div class="col power">
          <span class="status-dot" :class="serverStates[server.id]?.power || 'unknown'"></span>
          <span class="status-text">
            {{ 
              serverStates[server.id]?.power === 'checking' ? '检测中...' : 
              (serverStates[server.id]?.power === 'on' ? '已开机' : 
              (serverStates[server.id]?.power === 'off' ? '已关机' : '未知'))
            }}
          </span>
        </div>

        <!-- SSH 状态 -->
        <div class="col ssh-status">
          <template v-if="serverStates[server.id]?.power === 'on'">
            <span class="badge" :class="serverStates[server.id]?.ssh">
              {{ 
                serverStates[server.id]?.ssh === 'checking' ? '连接中...' : 
                (serverStates[server.id]?.ssh === 'online' ? '可连接' : '不可达') 
              }}
            </span>
          </template>
          <span v-else-if="serverStates[server.id]?.power === 'off'" class="text-muted">
            -
          </span>
          <span v-else class="text-muted">...</span>
        </div>

        <!-- 操作 -->
        <div class="col action action-col">
          <button 
            class="action-btn power-btn" 
            :class="serverStates[server.id]?.power === 'on' ? 'btn-danger' : 'btn-success'"
            @click="togglePower(server)"
            :disabled="serverStates[server.id]?.power === 'checking'"
          >
            {{ serverStates[server.id]?.power === 'on' ? '关机' : '开机' }}
          </button>
          
          <button 
            class="action-btn retry-btn" 
            @click="checkStatus(server)" 
            :disabled="serverStates[server.id]?.power === 'checking' || serverStates[server.id]?.ssh === 'checking'"
          >
            重试
          </button>
        </div>
      </div>
    </div>

    <!-- 确认对话框 Modal -->
    <div v-if="showConfirmModal" class="modal-overlay">
      <div class="modal-content">
        <h3>确认操作</h3>
        <p>确定要对 <strong>{{ pendingAction?.server.name }}</strong> 执行 <strong>{{ pendingAction?.actionType === 'on' ? '开机' : '关机' }}</strong> 操作吗？</p>
        <div class="modal-actions">
          <button @click="cancelAction" class="modal-btn cancel">取消</button>
          <button 
            @click="confirmAction" 
            class="modal-btn confirm" 
            :class="pendingAction?.actionType === 'on' ? 'confirm-on' : 'confirm-off'"
          >
            确定{{ pendingAction?.actionType === 'on' ? '开机' : '关机' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.monitor-container {
  padding: 0;
}

.header {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 15px;
}

.refresh-btn {
  padding: 8px 16px;
  background-color: #2196F3;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.refresh-btn:hover {
  background-color: #1976D2;
}

.server-list {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  overflow: hidden;
}

.list-header {
  display: grid;
  grid-template-columns: 2fr 2fr 1.5fr 1.5fr 2fr;
  padding: 15px 20px;
  background: #f5f5f5;
  font-weight: bold;
  color: #666;
  border-bottom: 1px solid #eee;
}

.list-item {
  display: grid;
  grid-template-columns: 2fr 2fr 1.5fr 1.5fr 2fr;
  padding: 15px 20px;
  border-bottom: 1px solid #eee;
  align-items: center;
}

.list-item:last-child {
  border-bottom: none;
}

/* Power Status Styles */
.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
  background-color: #ccc;
}

.status-dot.on { background-color: #4CAF50; box-shadow: 0 0 4px #4CAF50; }
.status-dot.off { background-color: #9e9e9e; }
.status-dot.checking { background-color: #ff9800; animation: blink 1s infinite; }

@keyframes blink {
  50% { opacity: 0.5; }
}

/* SSH Badge Styles */
.badge {
  padding: 4px 8px;
  border-radius: 12px;
  font-size: 0.85em;
  font-weight: 500;
}

.badge.online { background-color: #e8f5e9; color: #2e7d32; }
.badge.offline { background-color: #ffebee; color: #c62828; }
.badge.checking { background-color: #e3f2fd; color: #1565c0; }

.text-muted { color: #ccc; }

/* Action Buttons */
.action-col {
  display: flex;
  gap: 8px;
}

.action-btn {
  padding: 4px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9em;
  background: white;
  transition: all 0.2s;
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.retry-btn:hover:not(:disabled) {
  background: #f5f5f5;
  border-color: #ccc;
}

.btn-success {
  color: #2e7d32;
  border-color: #a5d6a7;
  background-color: #e8f5e9;
}

.btn-success:hover:not(:disabled) {
  background-color: #c8e6c9;
}

.btn-danger {
  color: #c62828;
  border-color: #ef9a9a;
  background-color: #ffebee;
}

.btn-danger:hover:not(:disabled) {
  background-color: #ffcdd2;
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.modal-content {
  background: white;
  padding: 24px;
  border-radius: 12px;
  width: 400px;
  max-width: 90%;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
  text-align: center;
  animation: modal-in 0.3s ease-out;
}

@keyframes modal-in {
  from { opacity: 0; transform: translateY(-20px); }
  to { opacity: 1; transform: translateY(0); }
}

.modal-content h3 {
  margin-top: 0;
  color: #333;
  font-size: 1.2rem;
}

.modal-content p {
  color: #666;
  margin: 20px 0;
  line-height: 1.5;
}

.modal-actions {
  margin-top: 24px;
  display: flex;
  justify-content: center;
  gap: 12px;
}

.modal-btn {
  padding: 8px 24px;
  border-radius: 6px;
  border: 1px solid #ddd;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.2s;
}

.modal-btn.cancel {
  background: white;
  color: #666;
}

.modal-btn.cancel:hover {
  background: #f5f5f5;
}

.modal-btn.confirm {
  color: white;
  border: none;
}

.modal-btn.confirm-on {
  background-color: #4CAF50;
}
.modal-btn.confirm-on:hover {
  background-color: #45a049;
  box-shadow: 0 2px 8px rgba(76, 175, 80, 0.3);
}

.modal-btn.confirm-off {
  background-color: #f44336;
}
.modal-btn.confirm-off:hover {
  background-color: #d32f2f;
  box-shadow: 0 2px 8px rgba(244, 67, 54, 0.3);
}
</style>
