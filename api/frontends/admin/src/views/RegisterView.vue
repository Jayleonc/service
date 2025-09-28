<template>
  <div class="auth-page">
    <el-card class="auth-card" shadow="hover">
      <h2 class="auth-card__title">注册</h2>
      <el-form ref="formRef" :model="form" :rules="rules" label-position="top" @submit.prevent="onSubmit">
        <el-form-item label="姓名" prop="name">
          <el-input v-model="form.name" placeholder="请输入姓名" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="请输入邮箱" type="email" autocomplete="username" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" placeholder="至少 8 位密码" autocomplete="new-password" show-password />
        </el-form-item>
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="form.phone" placeholder="可选" />
        </el-form-item>
        <el-alert v-if="error" type="error" :closable="false" class="auth-card__alert" :title="error" />
        <el-form-item>
          <el-button type="primary" :loading="loading" native-type="submit" class="auth-card__submit">注册</el-button>
        </el-form-item>
        <div class="auth-card__footer">
          <span>已经有账号？</span>
          <router-link to="/login">去登录</router-link>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/auth'

const store = useAuthStore()
const formRef = ref(null)
const form = reactive({
  name: '',
  email: '',
  password: '',
  phone: '',
})

const rules = {
  name: [{ required: true, message: '请输入姓名', trigger: 'blur' }],
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
    await store.register({ ...form })
    ElMessage.success('注册成功，请登录')
  } catch (err) {
    if (!err?.message) {
      ElMessage.error('注册失败，请稍后再试')
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
  width: min(480px, 100%);
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
