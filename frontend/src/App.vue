<script setup>
import { ref, onMounted } from 'vue'
import OpenVPNMonitor from './components/OpenVPNMonitor.vue'
import SSHMonitor from './components/SSHMonitor.vue'

// OpenVPN 网关配置
const vpnGateway = ref({
  id: 'vpn-gw',
  name: 'OpenVPN Gateway',
  ip: '10.8.0.1'
})

// 局域网内部服务器列表
const internalServers = ref([
  { id: 1, name: 'Web Server 01', ip: '192.168.1.101' },
  { id: 2, name: 'Database Server', ip: '192.168.1.102' },
  { id: 3, name: 'Backup Server', ip: '192.168.1.103' },
  { id: 5, name: 'Dev Environment', ip: '192.168.1.200' }
])

const vpnStatus = ref('checking') // 'checking', 'online', 'offline'

// 检查 VPN 状态
const checkVpnStatus = () => {
  vpnStatus.value = 'checking'
  // 模拟检查过程
  setTimeout(() => {
    // 模拟 90% 概率在线
    const isOnline = Math.random() > 0.1
    vpnStatus.value = isOnline ? 'online' : 'offline'
  }, 1500)
}

onMounted(() => {
  checkVpnStatus()
})
</script>

<template>
  <div class="app-container">
    <header class="main-header">
      <div class="logo">ServerMonitor</div>
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
        <div class="section-header">
          <h2>局域网服务器状态</h2>
        </div>
        <SSHMonitor :servers="internalServers" />
      </section>
      
      <div v-else-if="vpnStatus === 'offline'" class="offline-notice">
        <p>⚠️ OpenVPN 连接失败，无法访问内部服务器。</p>
      </div>
    </main>
  </div>
</template>

<style>
/* 全局重置 */
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
  background-color: #f0f2f5;
  color: #333;
}
</style>

<style scoped>
.app-container {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.main-header {
  background-color: #fff;
  padding: 0 20px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  display: flex;
  align-items: center;
  height: 60px;
  z-index: 10;
}

.logo {
  font-size: 1.2rem;
  font-weight: bold;
  color: #2c3e50;
}

.content-area {
  flex: 1;
  padding: 20px;
  max-width: 1000px;
  margin: 0 auto;
  width: 100%;
  box-sizing: border-box;
}

.section-header {
  margin-bottom: 15px;
  border-bottom: 2px solid #eee;
  padding-bottom: 10px;
}

.section-header h2 {
  margin: 0;
  font-size: 1.2rem;
  color: #444;
}

.offline-notice {
  text-align: center;
  padding: 40px;
  color: #666;
  background: #fff;
  border-radius: 8px;
  margin-top: 20px;
}
</style>
