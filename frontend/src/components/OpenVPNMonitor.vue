<script setup lang="ts">
interface Server {
  name: string;
  ip?: string;
  [key: string]: any;
}

defineProps<{
  server: Server;
  status?: string; // 'checking', 'online', 'offline', 'unknown'
}>()

defineEmits(['retry'])
</script>

<template>
  <div class="vpn-monitor-card" :class="status || 'unknown'">
    <div class="info-area">
      <div class="status-row">
        <div class="status-indicator-wrapper">
          <div class="status-indicator"></div>
          <div class="status-ping" v-if="status === 'online' || status === 'checking'"></div>
        </div>
        <h3>{{ server.name }}</h3>
      </div>
      <p class="status-message">
        <span v-if="status === 'checking'">正在检测连接状态...</span>
        <span v-else-if="status === 'online'">连接正常，局域网可访问</span>
        <span v-else-if="status === 'offline'">连接失败，无法访问内部网络</span>
        <span v-else>状态未知</span>
      </p>
    </div>
    <div class="action-area">
      <button 
        @click="$emit('retry')" 
        class="retry-btn"
        :disabled="status === 'checking'"
      >
        <span class="btn-icon" :class="{ spinning: status === 'checking' }">↻</span>
        {{ status === 'checking' ? '检测中...' : '刷新状态' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.vpn-monitor-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: 20px 24px;
  box-shadow: var(--shadow-md);
  margin-bottom: 32px;
  border: 1px solid var(--border-color);
  position: relative;
  overflow: hidden;
  transition: all 0.3s ease;
}

.vpn-monitor-card::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  background-color: var(--text-secondary);
  transition: background-color 0.3s ease;
}

.vpn-monitor-card.online::before { background-color: var(--success-color); }
.vpn-monitor-card.offline::before { background-color: var(--danger-color); }
.vpn-monitor-card.checking::before { background-color: var(--warning-color); }

.info-area {
  flex: 1;
}

.status-row {
  display: flex;
  align-items: center;
  margin-bottom: 6px;
}

.status-indicator-wrapper {
  position: relative;
  width: 12px;
  height: 12px;
  margin-right: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--text-secondary);
  z-index: 2;
  transition: background-color 0.3s ease;
}

.status-ping {
  position: absolute;
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background-color: inherit;
  opacity: 0.75;
  animation: ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite;
}

.online .status-indicator { background-color: var(--success-color); }
.online .status-ping { background-color: var(--success-color); }

.offline .status-indicator { background-color: var(--danger-color); }

.checking .status-indicator { background-color: var(--warning-color); }
.checking .status-ping { background-color: var(--warning-color); }

@keyframes ping {
  75%, 100% {
    transform: scale(2);
    opacity: 0;
  }
}

.info-area h3 {
  margin: 0;
  font-size: 1.125rem;
  color: var(--text-main);
  font-weight: 600;
}

.status-message {
  margin: 0;
  font-size: 0.875rem;
  color: var(--text-secondary);
  padding-left: 24px;
}

.action-area {
  margin-left: 24px;
}

.retry-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background-color: white;
  color: var(--text-main);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
  box-shadow: var(--shadow-sm);
}

.retry-btn:hover:not(:disabled) {
  background-color: var(--bg-page);
  border-color: var(--text-secondary);
  transform: translateY(-1px);
}

.retry-btn:active:not(:disabled) {
  transform: translateY(0);
}

.retry-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background-color: var(--bg-page);
}

.btn-icon {
  font-size: 1.1em;
  line-height: 1;
}

.spinning {
  animation: spin 1s linear infinite;
  display: inline-block;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
