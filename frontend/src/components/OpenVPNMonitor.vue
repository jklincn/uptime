<script setup>
defineProps({
  server: {
    type: Object,
    required: true
  },
  status: {
    type: String,
    default: 'unknown' // 'checking', 'online', 'offline', 'unknown'
  }
})

defineEmits(['retry'])
</script>

<template>
  <div class="vpn-monitor-card" :class="status">
    <div class="info-area">
      <div class="status-row">
        <div class="status-indicator"></div>
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
  background: white;
  border-radius: 8px;
  padding: 16px 24px;
  box-shadow: 0 2px 6px rgba(0,0,0,0.05);
  margin-bottom: 20px;
  border: 1px solid #eee;
  border-left: 4px solid #ccc;
}

.vpn-monitor-card.online {
  border-left-color: #4CAF50;
}

.vpn-monitor-card.offline {
  border-left-color: #f44336;
}

.vpn-monitor-card.checking {
  border-left-color: #ff9800;
}

.info-area {
  flex: 1;
}

.status-row {
  display: flex;
  align-items: center;
  margin-bottom: 4px;
}

.status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: #ccc;
  margin-right: 10px;
}

.online .status-indicator { background-color: #4CAF50; }
.offline .status-indicator { background-color: #f44336; }
.checking .status-indicator { background-color: #ff9800; }

.info-area h3 {
  margin: 0;
  font-size: 1.1rem;
  color: #333;
  font-weight: 600;
}

.status-message {
  margin: 0;
  font-size: 0.9rem;
  color: #666;
  padding-left: 20px; /* Align with text above (10px width + 10px margin) */
}

.action-area {
  margin-left: 20px;
}

.retry-btn {
  padding: 6px 16px;
  border: 1px solid #ddd;
  border-radius: 4px;
  background-color: white;
  color: #333;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.2s;
}

.retry-btn:hover:not(:disabled) {
  background-color: #f5f5f5;
  border-color: #ccc;
}

.retry-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
