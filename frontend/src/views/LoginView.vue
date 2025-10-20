<template>
  <div class="login-container">
    <el-card class="login-card">
      <h2 class="title">用户登录</h2>
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="example@domain.com" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" />
        </el-form-item>
        <div class="actions">
          <div class="register-link">
            <span>没有账号？</span>
            <router-link to="/register">立即注册</router-link>
          </div>
          <el-button type="primary" :loading="loading" @click="handleLogin">登录</el-button>
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
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: ['blur', 'change'] },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6个字符', trigger: 'blur' },
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
      ElMessage.success('登录成功');
    } catch (error) {
      console.error(error);
      ElMessage.error(error.response?.data?.message || '登录失败，请检查邮箱和密码');
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
  justify-content: space-between;
  align-items: center;
}

.register-link {
  font-size: 14px;
}

.register-link a {
  color: #409EFF;
  text-decoration: none;
  margin-left: 5px;
}

.register-link a:hover {
  text-decoration: underline;
}
</style>
