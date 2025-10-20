<template>
  <div class="register-container">
    <el-card class="register-card">
      <h2 class="title">注册账号</h2>
      <el-form :model="form" :rules="rules" ref="formRef" label-position="top">
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="example@domain.com" />
        </el-form-item>
        <el-form-item label="姓名" prop="name">
          <el-input v-model="form.name" placeholder="请输入您的姓名" />
        </el-form-item>
        <el-form-item label="手机号码" prop="phone">
          <el-input v-model="form.phone" placeholder="请输入您的手机号码" />
        </el-form-item>
        <el-form-item label="密码" prop="password">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input v-model="form.confirmPassword" type="password" show-password />
        </el-form-item>
        <div class="actions">
          <el-button @click="goToLogin">返回登录</el-button>
          <el-button type="primary" :loading="loading" @click="handleRegister">注册</el-button>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';

import { register } from '@/api/auth';

const router = useRouter();
const formRef = ref();
const loading = ref(false);
const form = reactive({
  email: '',
  name: '',
  phone: '',
  password: '',
  confirmPassword: '',
});

const validateConfirmPassword = (rule, value, callback) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'));
  } else {
    callback();
  }
};

const rules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: ['blur', 'change'] },
  ],
  name: [
    { required: true, message: '请输入姓名', trigger: 'blur' },
  ],
  phone: [
    { required: true, message: '请输入手机号码', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6个字符', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: ['blur', 'change'] },
  ],
};

const handleRegister = () => {
  formRef.value.validate(async (valid) => {
    if (!valid) {
      return;
    }
    loading.value = true;
    try {
      await register({
        email: form.email,
        name: form.name,
        phone: form.phone,
        password: form.password,
      });
      ElMessage.success('注册成功，请登录');
      router.push('/login');
    } catch (error) {
      console.error(error);
      ElMessage.error(error.response?.data?.message || '注册失败');
    } finally {
      loading.value = false;
    }
  });
};

const goToLogin = () => {
  router.push('/login');
};
</script>

<style scoped>
.register-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: linear-gradient(135deg, #5c6bc0, #3949ab);
}

.register-card {
  width: 400px;
}

.title {
  text-align: center;
  margin-bottom: 24px;
}

.actions {
  display: flex;
  justify-content: space-between;
  margin-top: 16px;
}
</style>
