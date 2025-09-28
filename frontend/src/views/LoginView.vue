<template>
  <div class="login-container">
    <el-card class="login-card">
      <h2 class="title">Service Console</h2>
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
        <el-form-item label="Email" prop="email">
          <el-input v-model="form.email" placeholder="example@domain.com" />
        </el-form-item>
        <el-form-item label="Password" prop="password">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <div class="actions">
          <el-button type="primary" :loading="loading" @click="handleLogin">Login</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { ElMessage } from 'element-plus';

import { useAuthStore } from '@/store/auth';

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const formRef = ref();
const loading = ref(false);
const form = reactive({
  email: '',
  password: '',
});

const rules = {
  email: [
    { required: true, message: 'Email is required', trigger: 'blur' },
    { type: 'email', message: 'Enter a valid email', trigger: ['blur', 'change'] },
  ],
  password: [
    { required: true, message: 'Password is required', trigger: 'blur' },
  ],
};

const handleLogin = () => {
  formRef.value.validate(async (valid) => {
    if (!valid) {
      return;
    }
    loading.value = true;
    try {
      await auth.login({ email: form.email, password: form.password });
      const redirect = route.query.redirect || (auth.isAdmin ? '/admin/users' : '/me');
      router.replace(redirect);
      ElMessage.success('Login successful');
    } catch (error) {
      console.error(error);
      ElMessage.error(error.response?.data?.message || 'Failed to login');
    } finally {
      loading.value = false;
    }
  });
};
</script>

<style scoped>
.login-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #5c6bc0, #3949ab);
}

.login-card {
  width: 360px;
}

.title {
  text-align: center;
  margin-bottom: 24px;
}

.actions {
  display: flex;
  justify-content: flex-end;
}
</style>
