<template>
  <v-container class="fill-height" fluid>
    <v-row align="center" justify="center">
      <v-col cols="12" sm="8" md="4">
        <v-card>
          <v-card-title>注册</v-card-title>
          <v-card-text>
            <v-form @submit.prevent="onSubmit">
              <v-text-field v-model="name" label="姓名" required />
              <v-text-field v-model="email" label="邮箱" type="email" required />
              <v-text-field v-model="phone" label="电话" />
              <v-text-field v-model="password" label="密码" type="password" required />
              <v-alert type="error" v-if="error" class="mb-2">{{ error }}</v-alert>
              <v-btn :loading="loading" type="submit" color="primary" block>注册</v-btn>
              <v-btn variant="text" block to="/login">已有账号？去登录</v-btn>
            </v-form>
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useAuthStore } from '@/store/auth'

const store = useAuthStore()
const name = ref('')
const email = ref('')
const phone = ref('')
const password = ref('')
const loading = computed(() => store.loading)
const error = computed(() => store.error)

async function onSubmit () {
  await store.register({ name: name.value, email: email.value, password: password.value, phone: phone.value })
}
</script>
