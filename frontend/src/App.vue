<script setup>
import { ref, onMounted } from 'vue'
import OpenVPNMonitor from './components/OpenVPNMonitor.vue'
import SSHMonitor from './components/SSHMonitor.vue'

// OpenVPN 网关配置
const vpnGateway = ref({
  id: 'vpn-gw',
  name: 'OpenVPN Gateway',
  ip: '10.183.111.1'
})

// 局域网内部服务器列表
const internalServers = ref([])

const fetchServers = async () => {
  try {
    const response = await fetch('http://localhost:23080/api/servers')
    if (response.ok) {
      internalServers.value = await response.json()
    } else {
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
    // 1. 检查 VPN 连接状态 (Ping 10.183.111.1)
    const vpnRes = await fetch('http://localhost:23080/api/vpn/status')
    if (vpnRes.ok) {
      const data = await vpnRes.json()
      vpnStatus.value = data.status // 'online' or 'offline'
      
      // 2. 如果 VPN 在线，获取服务器列表
      if (vpnStatus.value === 'online') {
        fetchServers()
      }
    } else {
      vpnStatus.value = 'offline'
    }
  } catch (error) {
    console.error('Connection check failed:', error)
    vpnStatus.value = 'offline'
  }
}

onMounted(() => {
  checkVpnStatus()
})
</script>

<template>
  <div class="app-container">
    <header class="main-header">
      <div class="header-content">
        <div class="logo">ServerMonitor</div>
      </div>
    </header>

    <main class="content-area">
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
