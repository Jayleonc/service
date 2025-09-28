<template>
  <div class="auth-page">
    <el-card class="auth-card" shadow="hover">
      <h2 class="auth-card__title">登录</h2>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" type="email" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="请输入密码" autocomplete="current-password" show-password />
        </el-form-item>
        <el-alert v-if="error" type="error" :closable="false" class="auth-card__alert" :title="error" />
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" class="auth-card__submit">登录</el-button>
        </el-form-item>
        <div class="auth-card__footer">
          <span>还没有账号？</span>
          <router-link to="/register">立即注册</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/auth'

const route = useRoute()
const store = useAuthStore()

const formRef = ref(null)
const form = reactive({
  email: '',
  password: '',
})

const rules = {
  email: [
    { required: true, message: '请输入邮箱地址', trigger: 'blur' },
    { type: 'email', message: '邮箱格式不正确', trigger: ['blur', 'change'] },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, message: '密码至少 8 位', trigger: 'blur' },
  ],
}

const loading = computed(() => store.loading)
const error = computed(() => store.error)

async function onSubmit() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch (e) {
    return
  }
  try {
    const redirect = route.query.redirect
    await store.login(form.email, form.password, redirect)
    ElMessage.success('登录成功')
  } catch (err) {
    if (!err?.message) {
      ElMessage.error('登录失败，请稍后再试')
    }
  }
}
</script>

<style scoped>
.auth-page {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  padding: 24px;
}

.auth-card {
  width: min(420px, 100%);
  border-radius: 16px;
}

.auth-card__title {
  margin: 0 0 16px;
  font-size: 24px;
  font-weight: 600;
  text-align: center;
  color: #1f2937;
}

.auth-card__alert {
  margin-bottom: 16px;
}

.auth-card__submit {
  width: 100%;
}

.auth-card__footer {
  display: flex;
  justify-content: center;
  gap: 8px;
  color: #475569;
}
</style>
