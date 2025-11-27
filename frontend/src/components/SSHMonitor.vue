<script setup>
import { ref, onMounted } from 'vue'

const props = defineProps({
  servers: {
    type: Array,
    required: true
  }
})

// 存储每个服务器的状态：{ power: 'on'|'off'|'checking'|'unknown', network: 'online'|'offline'|'checking'|'waiting'|'skipped' }
const serverStates = ref({})

// 确认对话框状态
const showConfirmModal = ref(false)
const pendingAction = ref(null) // { server: object, actionType: 'on'|'off' }
const actionStatus = ref('confirm') // 'confirm' | 'executing' | 'success' | 'error' | 'timeout'
const actionMessage = ref('')

const checkStatus = async (server) => {
  // 初始化状态
  serverStates.value[server.name] = { power: 'checking', network: 'waiting' }
  
  // 1. 模拟检查电源状态 (BMC)
  await new Promise(resolve => setTimeout(resolve, 600 + Math.random() * 600))
  
  // 模拟: 70% 开机, 20% 关机, 10% 无法获取
  const rand = Math.random()
  let powerStatus = 'on'
  if (rand > 0.9) powerStatus = 'unknown'
  else if (rand > 0.7) powerStatus = 'off'
  
  // 更新电源状态
  serverStates.value[server.name].power = powerStatus
  
  // 2. 如果开机 或 无法获取电源状态，继续检查网络连通性
  if (powerStatus === 'on' || powerStatus === 'unknown') {
    serverStates.value[server.name].network = 'checking'
    
    await new Promise(resolve => setTimeout(resolve, 600 + Math.random() * 1000))
    
    // 模拟 90% 网络可达
    const isNetworkOnline = Math.random() > 0.1
    serverStates.value[server.name].network = isNetworkOnline ? 'online' : 'offline'
  } else {
    serverStates.value[server.name].network = 'skipped'
  }
}

const togglePower = (server) => {
  const currentState = serverStates.value[server.name]?.power
  if (!currentState || currentState === 'checking' || currentState === 'unknown') return

  const actionType = currentState === 'on' ? 'off' : 'on'
  pendingAction.value = { server, actionType }
  actionStatus.value = 'confirm'
  actionMessage.value = ''
  showConfirmModal.value = true
}

const confirmAction = async () => {
  if (!pendingAction.value) return
  
  const { server, actionType } = pendingAction.value
  actionStatus.value = 'executing'

  // 模拟 API 调用
  try {
    // 模拟 3秒执行时间
    await new Promise((resolve, reject) => {
      setTimeout(() => {
        // 模拟 10% 概率超时/失败
        if (Math.random() > 0.9) {
          reject(new Error('timeout'))
        } else {
          resolve()
        }
      }, 2000 + Math.random() * 1000)
    })

    actionStatus.value = 'success'
    actionMessage.value = `${actionType === 'on' ? '开机' : '关机'}指令已发送成功`
    
    // 更新本地状态为检测中
    serverStates.value[server.name].power = 'checking'
    serverStates.value[server.name].network = 'waiting'

  } catch (error) {
    if (error.message === 'timeout') {
      actionStatus.value = 'timeout'
      actionMessage.value = '操作超时，请稍后重试'
    } else {
      actionStatus.value = 'error'
      actionMessage.value = '操作失败，请检查网络或权限'
    }
  }
}

const closeResultModal = () => {
  const { server } = pendingAction.value || {}
  showConfirmModal.value = false
  pendingAction.value = null
  actionStatus.value = 'confirm'
  
  // 如果操作成功，关闭弹窗后刷新状态
  if (server) {
    checkStatus(server)
  }
}

const cancelAction = () => {
  showConfirmModal.value = false
  pendingAction.value = null
  actionStatus.value = 'confirm'
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
      <h2>局域网服务器状态</h2>
      <button @click="checkAll" class="refresh-btn">
        <span class="btn-icon">↻</span> 刷新列表状态
      </button>
    </div>
    
    <div class="server-list">
      <div class="list-header">
        <div class="col">服务器名称</div>
        <div class="col">IP 地址</div>
        <div class="col">电源状态</div>
        <div class="col">网络状态</div>
        <div class="col action-col">操作</div>
      </div>
      
      <transition-group name="list" tag="div">
        <div v-for="server in servers" :key="server.name" class="list-item">
          <div class="col name">{{ server.name }}</div>
          <div class="col ip">{{ server.ip }}</div>
          
          <!-- 电源状态 -->
          <div class="col power">
            <span class="status-dot" :class="serverStates[server.name]?.power || 'unknown'"></span>
            <span class="status-text">
              {{ 
                serverStates[server.name]?.power === 'checking' ? '检测中...' : 
                (serverStates[server.name]?.power === 'on' ? '已开机' : 
                (serverStates[server.name]?.power === 'off' ? '已关机' : '无法获取'))
              }}
            </span>
          </div>

          <!-- 网络状态 -->
          <div class="col network-status">
            <template v-if="serverStates[server.name]?.power === 'on' || serverStates[server.name]?.power === 'unknown'">
              <span class="badge" :class="serverStates[server.name]?.network">
                {{ 
                  serverStates[server.name]?.network === 'checking' ? '检测中...' : 
                  (serverStates[server.name]?.network === 'online' ? '在线' : '不可达') 
                }}
              </span>
            </template>
            <span v-else-if="serverStates[server.name]?.power === 'off'" class="text-muted">
              -
            </span>
            <span v-else class="text-muted">...</span>
          </div>

          <!-- 操作 -->
          <div class="col action action-col">
            <button 
              class="action-btn power-btn" 
              :class="serverStates[server.name]?.power === 'on' ? 'btn-danger' : (serverStates[server.name]?.power === 'off' ? 'btn-success' : 'btn-disabled')"
              @click="togglePower(server)"
              :disabled="serverStates[server.name]?.power === 'checking' || serverStates[server.name]?.power === 'unknown'"
              :title="serverStates[server.name]?.power === 'on' ? '关机' : (serverStates[server.name]?.power === 'off' ? '开机' : '无法操作')"
            >
              {{ serverStates[server.name]?.power === 'on' ? '关机' : (serverStates[server.name]?.power === 'off' ? '开机' : '无法操作') }}
            </button>
            
            <button 
              class="action-btn retry-btn" 
              @click="checkStatus(server)" 
              :disabled="serverStates[server.name]?.power === 'checking' || serverStates[server.name]?.network === 'checking'"
              title="重试连接"
            >
              重试
            </button>
          </div>
        </div>
      </transition-group>
    </div>

    <!-- 确认对话框 Modal -->
    <transition name="modal">
      <div v-if="showConfirmModal" class="modal-overlay">
        <div class="modal-content">
          <!-- 标题始终显示 -->
          <h3 :class="actionStatus === 'success' ? 'success' : (actionStatus === 'error' || actionStatus === 'timeout' ? 'error' : '')">
            {{ 
              actionStatus === 'confirm' ? '确认操作' : 
              (actionStatus === 'executing' ? '正在执行...' : 
              (actionStatus === 'success' ? '操作成功' : '操作失败')) 
            }}
          </h3>

          <!-- 内容区域固定高度 -->
          <div class="modal-body">
            <!-- 确认阶段 -->
            <template v-if="actionStatus === 'confirm'">
              <div class="confirm-icon">!</div>
              <p>确定要对 <strong class="server-name">{{ pendingAction?.server.name }}</strong> 执行 <strong :class="pendingAction?.actionType === 'on' ? 'text-success' : 'text-danger'">{{ pendingAction?.actionType === 'on' ? '开机' : '关机' }}</strong> 操作吗？</p>
            </template>

            <!-- 执行中阶段 -->
            <template v-else-if="actionStatus === 'executing'">
              <div class="loading-spinner"></div>
              <p>正在发送{{ pendingAction?.actionType === 'on' ? '开机' : '关机' }}指令，请稍候...</p>
            </template>

            <!-- 结果阶段 -->
            <template v-else>
              <div class="result-icon" :class="actionStatus">
                {{ actionStatus === 'success' ? '✓' : '!' }}
              </div>
              <p>{{ actionMessage }}</p>
            </template>
          </div>

          <!-- 按钮区域 -->
          <div class="modal-actions">
            <template v-if="actionStatus === 'confirm'">
              <button @click="cancelAction" class="modal-btn cancel">取消</button>
              <button 
                @click="confirmAction" 
                class="modal-btn confirm" 
                :class="pendingAction?.actionType === 'on' ? 'confirm-on' : 'confirm-off'"
              >
                确定{{ pendingAction?.actionType === 'on' ? '开机' : '关机' }}
              </button>
            </template>
            <template v-else-if="actionStatus === 'executing'">
              <!-- 执行中不显示按钮，或者显示禁用按钮 -->
            </template>
            <template v-else>
              <button @click="closeResultModal" class="modal-btn primary">关闭</button>
            </template>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<style scoped>
/* ...existing code... */
.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid var(--primary-color);
  border-radius: 50%;
  margin: 0 0 20px 0;
  animation: spin 1s linear infinite;
}

.result-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 0 20px 0;
  font-size: 24px;
  font-weight: bold;
  color: white;
}

.result-icon.success {
  background-color: var(--success-color);
}

.result-icon.error, .result-icon.timeout {
  background-color: var(--danger-color);
}

.confirm-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 0 20px 0;
  font-size: 24px;
  font-weight: bold;
  color: #d97706; /* Amber 600 */
  background-color: #fffbeb; /* Amber 50 */
  border: 1px solid #fcd34d; /* Amber 300 */
}

.server-name {
  color: var(--text-main);
  font-size: 1.1em;
}

.text-success { color: var(--success-color); }
.text-danger { color: var(--danger-color); }

h3.success { color: var(--success-color); }
h3.error, h3.timeout { color: var(--danger-color); }

.modal-btn.primary {
  background-color: var(--primary-color);
  color: white;
  border: none;
}
.modal-btn.primary:hover {
  background-color: var(--primary-hover);
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.monitor-container {
  padding: 0;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.header h2 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-main);
}

.refresh-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background-color: white;
  color: var(--text-main);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all 0.2s;
  font-size: 0.875rem;
  font-weight: 500;
  box-shadow: var(--shadow-sm);
}

.refresh-btn:hover {
  background-color: var(--bg-page);
  border-color: var(--text-secondary);
  transform: translateY(-1px);
}

.refresh-btn:active {
  transform: translateY(0);
}

.btn-icon {
  font-size: 1.1em;
  line-height: 1;
}

.server-list {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  overflow: hidden;
  border: 1px solid var(--border-color);
}

.list-header {
  display: grid;
  grid-template-columns: 2fr 2fr 1.5fr 1.5fr 2fr;
  padding: 16px 24px;
  background: var(--bg-page);
  font-weight: 600;
  color: var(--text-secondary);
  font-size: 0.875rem;
  border-bottom: 1px solid var(--border-color);
}

.list-item {
  display: grid;
  grid-template-columns: 2fr 2fr 1.5fr 1.5fr 2fr;
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-color);
  align-items: center;
  transition: background-color 0.2s;
}

.list-item:last-child {
  border-bottom: none;
}

.list-item:hover {
  background-color: #f9fafb;
}

.col {
  font-size: 0.9375rem;
}

.col.name {
  font-weight: 500;
  color: var(--text-main);
}

.col.ip {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  color: var(--text-secondary);
  font-size: 0.85rem;
  letter-spacing: 0.02em;
}

/* Power Status Styles */
.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 8px;
  background-color: var(--text-secondary);
}

.status-dot.on { background-color: var(--success-color); box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.2); }
.status-dot.off { background-color: var(--text-secondary); opacity: 0.5; }
.status-dot.unknown { background-color: var(--warning-color); opacity: 0.5; }
.status-dot.checking { background-color: var(--warning-color); animation: blink 1s infinite; }

@keyframes blink {
  50% { opacity: 0.5; }
}

/* SSH Badge Styles */
.badge {
  padding: 4px 10px;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.025em;
}

.badge.online { background-color: #d1fae5; color: #065f46; }
.badge.offline { background-color: #fee2e2; color: #991b1b; }
.badge.checking { background-color: #dbeafe; color: #1e40af; }

.text-muted { color: var(--text-secondary); opacity: 0.5; }

/* Action Buttons */
.action-col {
  display: flex;
  gap: 8px;
}

.action-btn {
  padding: 6px 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  font-size: 0.8125rem;
  font-weight: 500;
  background: white;
  transition: all 0.2s;
  color: var(--text-main);
}

.action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  background-color: var(--bg-page);
}

.retry-btn:hover:not(:disabled) {
  background: var(--bg-page);
  border-color: var(--text-secondary);
}

.btn-success {
  color: var(--success-color);
  border-color: #d1fae5;
  background-color: #ecfdf5;
}

.btn-success:hover:not(:disabled) {
  background-color: #d1fae5;
  border-color: var(--success-color);
}

.btn-danger {
  color: var(--danger-color);
  border-color: #fee2e2;
  background-color: #fef2f2;
}

.btn-danger:hover:not(:disabled) {
  background-color: #fee2e2;
  border-color: var(--danger-color);
}

.btn-disabled {
  color: var(--text-secondary);
  border-color: var(--border-color);
  background-color: var(--bg-page);
  cursor: not-allowed;
}

/* List Transitions */
.list-enter-active,
.list-leave-active {
  transition: all 0.3s ease;
}
.list-enter-from,
.list-leave-to {
  opacity: 0;
  transform: translateX(-10px);
}

/* Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.4);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal-content {
  background: var(--bg-card);
  padding: 40px 48px;
  border-radius: var(--radius-lg);
  width: 480px;
  max-width: 90%;
  min-height: 320px;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  align-items: center;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  text-align: center;
  box-sizing: border-box;
}

.modal-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 140px;
}

.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active .modal-content,
.modal-leave-active .modal-content {
  transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.modal-enter-from .modal-content,
.modal-leave-to .modal-content {
  transform: scale(0.95) translateY(10px);
}

.modal-content h3 {
  margin: 0 0 24px 0;
  color: var(--text-main);
  font-size: 1.25rem;
  font-weight: 600;
  flex-shrink: 0;
  height: 28px;
  line-height: 28px;
}

.modal-content p {
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.6;
  width: 100%;
}

.modal-actions {
  display: flex;
  justify-content: center;
  gap: 16px;
  margin-top: auto;
  flex-shrink: 0;
  height: 44px;
  width: 100%;
  align-items: center;
}

.modal-btn {
  padding: 10px 24px;
  border-radius: var(--radius-md);
  border: 1px solid transparent;
  cursor: pointer;
  font-size: 0.9375rem;
  font-weight: 500;
  transition: all 0.2s;
}

.modal-btn.cancel {
  background: white;
  color: var(--text-main);
  border-color: var(--border-color);
}

.modal-btn.cancel:hover {
  background: var(--bg-page);
  border-color: var(--text-secondary);
}

.modal-btn.confirm {
  color: white;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
}

.modal-btn.confirm:active {
  transform: translateY(1px);
}

.modal-btn.confirm-on {
  background-color: var(--success-color);
}
.modal-btn.confirm-on:hover {
  background-color: #059669; /* Emerald 600 */
}

.modal-btn.confirm-off {
  background-color: var(--danger-color);
}
.modal-btn.confirm-off:hover {
  background-color: #dc2626; /* Red 600 */
}
</style>
