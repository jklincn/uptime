<script setup lang="ts">
import { ref, onMounted } from 'vue'
import OpenVPNMonitor from '../components/OpenVPNMonitor.vue'
import SSHMonitor from '../components/SSHMonitor.vue'

// 认证状态
const isAuthenticated = ref(false)
const isCheckingAuth = ref(!!localStorage.getItem('authToken')) // 初始化检查状态
const authForm = ref({
  phone: '',
  code: ''
})
const authStatus = ref('') // 'sending', 'sent', 'verifying', 'error'
const authMessage = ref('')

// OpenVPN 网关配置
const vpnGateway = ref({
  id: 'vpn-gw',
  name: 'OpenVPN Gateway', // 默认名称，会被后端配置覆盖
})

// 局域网内部服务器列表
const internalServers = ref<any[]>([])
const backendStatus = ref('checking') // 'checking', 'online', 'offline'

const getAuthHeaders = (): Record<string, string> => {
  const token = localStorage.getItem('authToken')
  const headers: Record<string, string> = {}
  if (token) {
    headers['Authorization'] = token
  }
  return headers
}

const checkBackendStatus = async () => {
  try {
    const res = await fetch('/api/ping')
    if (res.ok) {
      backendStatus.value = 'online'
    } else {
      backendStatus.value = 'offline'
    }
  } catch (e) {
    backendStatus.value = 'offline'
  }
}

const sendCode = async () => {
  if (!authForm.value.phone) {
    authMessage.value = '请输入手机号'
    return
  }
  
  authStatus.value = 'sending'
  authMessage.value = ''
  
  try {
    const res = await fetch('/api/auth/send-code', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phone: authForm.value.phone })
    })
    
    if (res.ok) {
      authStatus.value = 'sent'
      authMessage.value = '验证码已发送'
    } else {
      const data = await res.json()
      authStatus.value = 'error'
      authMessage.value = data.error || '发送失败'
    }
  } catch (e) {
    authStatus.value = 'error'
    authMessage.value = '网络错误'
  }
}

const login = async () => {
  if (!authForm.value.phone || !authForm.value.code) {
    authMessage.value = '请输入手机号和验证码'
    return
  }
  
  authStatus.value = 'verifying'
  
  try {
    const res = await fetch('/api/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        phone: authForm.value.phone,
        code: authForm.value.code
      })
    })
    
    if (res.ok) {
      const data = await res.json()
      localStorage.setItem('authToken', data.token)
      isAuthenticated.value = true
      checkVpnStatus()
    } else {
      const data = await res.json()
      authStatus.value = 'error'
      authMessage.value = data.error || '验证失败'
    }
  } catch (e) {
    authStatus.value = 'error'
    authMessage.value = '网络错误'
  }
}

const fetchServers = async () => {
  try {
    const response = await fetch('/api/servers', {
      headers: getAuthHeaders()
    })
    if (response.ok) {
      internalServers.value = await response.json()
    } else {
      if (response.status === 401) {
        isAuthenticated.value = false
        localStorage.removeItem('authToken')
      }
      console.error('Failed to fetch servers')
    }
  } catch (error) {
    console.error('Error fetching servers:', error)
  }
}

const vpnStatus = ref('checking') // 'checking', 'online', 'offline'

// 检查 VPN 状态
const checkVpnStatus = async () => {
  vpnStatus.value = 'checking'
  
  try {
    // 1. 检查 VPN 连接状态
    const vpnRes = await fetch('/api/vpn/status', {
      headers: getAuthHeaders()
    })
    
    if (vpnRes.ok) {
      const data = await vpnRes.json()
      vpnStatus.value = data.status // 'online' or 'offline'
      if (data.name) {
        vpnGateway.value.name = data.name
      }
      isAuthenticated.value = true // 确认认证成功
      
      // 2. 如果 VPN 在线，获取服务器列表
      if (vpnStatus.value === 'online') {
        fetchServers()
      }
    } else {
      if (vpnRes.status === 401) {
        isAuthenticated.value = false
        localStorage.removeItem('authToken')
      }
      vpnStatus.value = 'offline'
    }
  } catch (error) {
    console.error('Connection check failed:', error)
    vpnStatus.value = 'offline'
  } finally {
    isCheckingAuth.value = false // 检查完成
  }
}

onMounted(() => {
  checkBackendStatus()
  const token = localStorage.getItem('authToken')
  if (token) {
    // 不直接设置 isAuthenticated = true，而是等待 checkVpnStatus 验证
    checkVpnStatus()
  } else {
    isCheckingAuth.value = false
  }
})
</script>

<template>
  <div class="app-container">
    <header class="main-header">
      <div class="header-content">
        <div class="logo">ServerMonitor</div>
        <div class="backend-status" :class="backendStatus" title="后端服务状态">
          <span class="status-dot"></span>
          {{ backendStatus === 'online' ? '服务正常' : '服务离线' }}
        </div>
      </div>
    </header>

    <main class="content-area">
      <!-- 加载中状态 -->
      <div v-if="isCheckingAuth" class="loading-container">
        <div class="loading-spinner"></div>
        <p>正在验证身份...</p>
      </div>

      <!-- 登录认证界面 -->
      <div v-else-if="!isAuthenticated" class="auth-container">
        <div class="auth-card">
          <h2>登陆</h2>
          
          <div class="form-group">
            <label>手机号码</label>
            <div class="input-group">
              <input v-model="authForm.phone" type="text" placeholder="请输入手机号" />
              <button @click="sendCode" :disabled="authStatus === 'sending' || authStatus === 'sent'">
                {{ authStatus === 'sending' ? '发送中...' : (authStatus === 'sent' ? '已发送' : '获取验证码') }}
              </button>
            </div>
          </div>
          
          <div class="form-group">
            <label>验证码</label>
            <input v-model="authForm.code" type="text" placeholder="请输入验证码" />
          </div>
          
          <div v-if="authMessage" class="auth-message" :class="{ error: authStatus === 'error' }">
            {{ authMessage }}
          </div>
          
          <button class="login-btn" @click="login" :disabled="authStatus === 'verifying'">
            {{ authStatus === 'verifying' ? '验证中...' : '确认登录' }}
          </button>
        </div>
      </div>

      <!-- 主内容区域 -->
      <template v-else>
        <!-- OpenVPN 状态监控 -->
        <section class="vpn-section">
          <OpenVPNMonitor 
            :server="vpnGateway" 
            :status="vpnStatus" 
            @retry="checkVpnStatus"
          />
        </section>

        <!-- 内部服务器 SSH 监控 (仅当 VPN 在线时显示) -->
        <section class="ssh-section" v-if="vpnStatus === 'online'">
          <SSHMonitor :servers="internalServers" />
        </section>
        
        <div v-else-if="vpnStatus === 'offline'" class="offline-notice">
          <p>⚠️ OpenVPN 连接失败，无法访问内部服务器。</p>
        </div>
      </template>
    </main>
  </div>
</template>

<style>
/* 全局重置与变量 */
:root {
  --primary-color: #3b82f6; /* Blue 500 */
  --primary-hover: #2563eb; /* Blue 600 */
  --success-color: #10b981; /* Emerald 500 */
  --danger-color: #ef4444; /* Red 500 */
  --warning-color: #f59e0b; /* Amber 500 */
  --text-main: #1f2937; /* Gray 800 */
  --text-secondary: #6b7280; /* Gray 500 */
  --bg-page: #f3f4f6; /* Gray 100 */
  --bg-card: #ffffff;
  --border-color: #e5e7eb; /* Gray 200 */
  --shadow-sm: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
  --radius-md: 0.5rem;
  --radius-lg: 0.75rem;
}

body {
  margin: 0;
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
  background-color: var(--bg-page);
  color: var(--text-main);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

button {
  font-family: inherit;
}

/* 认证界面样式 */
.auth-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 60vh;
}

.auth-card {
  background: white;
  padding: 2rem;
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  width: 100%;
  max-width: 400px;
}

.auth-card h2 {
  margin-top: 0;
  margin-bottom: 0.5rem;
  text-align: center;
  color: var(--text-main);
}

.auth-desc {
  text-align: center;
  color: var(--text-secondary);
  margin-bottom: 2rem;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: var(--text-main);
}

.form-group input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  font-size: 1rem;
  box-sizing: border-box;
}

.input-group {
  display: flex;
  gap: 0.5rem;
}

.input-group input {
  flex: 1;
}

.input-group button {
  padding: 0 1rem;
  background: var(--bg-page);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  cursor: pointer;
  white-space: nowrap;
}

.login-btn {
  width: 100%;
  padding: 0.75rem;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  font-size: 1rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s;
}

.login-btn:hover {
  background: var(--primary-hover);
}

.login-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
}

.auth-message {
  margin-bottom: 1rem;
  padding: 0.5rem;
  border-radius: var(--radius-md);
  background: #f0fdf4;
  color: var(--success-color);
  text-align: center;
  font-size: 0.875rem;
}

.auth-message.error {
  background: #fef2f2;
  color: var(--danger-color);
}

/* 加载中状态样式 */
.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 60vh;
  color: var(--text-secondary);
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 4px solid #f3f3f3;
  border-top: 4px solid var(--primary-color);
  border-radius: 50%;
  margin-bottom: 16px;
  animation: spin 1s linear infinite;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
</style>

<style scoped>
.app-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.main-header {
  background-color: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid var(--border-color);
  height: 52px;
  position: sticky;
  top: 0;
  z-index: 50;
}

.header-content {
  max-width: 800px;
  margin: 0 auto;
  height: 100%;
  display: flex;
  align-items: center;
  padding: 0 24px;
  box-sizing: border-box;
}

.logo {
  font-size: 1.125rem;
  font-weight: 700;
  color: var(--text-main);
  letter-spacing: -0.025em;
  display: flex;
  align-items: center;
  gap: 8px;
}

.logo::before {
  content: '';
  display: block;
  width: 10px;
  height: 10px;
  background: var(--primary-color);
  border-radius: 3px;
  transform: rotate(45deg);
}

.backend-status {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.backend-status .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--text-secondary);
}

.backend-status.online .status-dot {
  background-color: var(--success-color);
  box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.2);
}

.backend-status.offline .status-dot {
  background-color: var(--danger-color);
}

.content-area {
  flex: 1;
  padding: 32px 24px;
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

.section-header {
  margin-bottom: 20px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-header h2 {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-main);
}

.ssh-section {
  animation: fadeIn 0.5s ease-out;
}

.offline-notice {
  text-align: center;
  padding: 48px;
  color: var(--text-secondary);
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  margin-top: 24px;
  box-shadow: var(--shadow-sm);
  border: 1px solid var(--border-color);
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
