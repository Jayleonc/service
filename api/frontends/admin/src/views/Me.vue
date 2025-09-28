<template>
  <v-container>
    <v-card>
      <v-card-title>我的信息</v-card-title>
      <v-card-text>
        <v-form @submit.prevent="onSave">
          <v-text-field v-model="name" label="姓名" />
          <v-text-field v-model="phone" label="电话" />
          <div class="text-caption mb-2">邮箱：{{ email }}</div>
          <v-btn :loading="saving" type="submit" color="primary">保存</v-btn>
          <v-btn class="ml-2" color="error" @click="onLogout">退出登录</v-btn>
        </v-form>
      </v-card-text>
    </v-card>
  </v-container>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/store/auth'

const store = useAuthStore()
const saving = ref(false)
const name = ref('')
const phone = ref('')
const email = computed(() => store.user?.email || '')

onMounted(async () => {
  if (!store.user) {
    await store.fetchMe()
  }
  name.value = store.user?.name || ''
  phone.value = store.user?.phone || ''
})

async function onSave () {
  saving.value = true
  try {
    await store.updateMe({ name: name.value, phone: phone.value })
  } finally {
    saving.value = false
  }
}

async function onLogout () {
  await store.logout()
}
</script>
