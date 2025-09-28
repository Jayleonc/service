<template>
  <el-container class="layout">
    <el-header class="layout__header">
      <div class="layout__title">Service Admin</div>
      <div class="layout__spacer" />
      <div class="layout__actions">
        <span v-if="user" class="layout__user">{{ user.name }}</span>
        <el-button link type="primary" @click="goProfile">个人信息</el-button>
        <el-divider direction="vertical" />
        <el-button type="danger" text @click="logout">退出</el-button>
      </div>
    </el-header>
    <el-main class="layout__main">
      <slot />
    </el-main>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/store/auth'

const router = useRouter()
const auth = useAuthStore()

const user = computed(() => auth.user)

function goProfile() {
  router.push({ name: 'Profile' })
}

async function logout() {
  await auth.logout()
}
</script>

<style scoped>
.layout {
  min-height: 100vh;
  background: transparent;
}

.layout__header {
  display: flex;
  align-items: center;
  padding: 0 32px;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(15, 23, 42, 0.08);
}

.layout__title {
  font-size: 1.25rem;
  font-weight: 600;
  color: #1f2937;
}

.layout__spacer {
  flex: 1 1 auto;
}

.layout__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #475569;
}

.layout__user {
  font-weight: 500;
}

.layout__main {
  padding: 32px;
  display: flex;
  justify-content: center;
}
</style>
