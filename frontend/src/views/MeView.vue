<template>
  <div class="profile-container">
    <el-card class="card">
      <template #header>
        <div class="card-header">
          <span class="header-title">个人信息</span>
          <el-button type="primary" :loading="loading" @click="save">保存修改</el-button>
        </div>
      </template>
      <div class="profile-content">
        <div class="avatar-section">
          <el-avatar :size="100" :src="avatarUrl" class="avatar">
            {{ form.name ? form.name.charAt(0).toUpperCase() : 'U' }}
          </el-avatar>
          <div class="user-name">{{ form.name || '未设置名称' }}</div>
        </div>
        <el-form :model="form" label-width="120px" class="form-section">
          <el-form-item label="名称">
            <el-input v-model="form.name" placeholder="请输入您的名称" />
          </el-form-item>
          <el-form-item label="邮箱">
            <el-input v-model="form.email" disabled />
          </el-form-item>
          <el-form-item label="手机号码">
            <el-input v-model="form.phone" placeholder="请输入您的手机号码" />
          </el-form-item>
          <el-form-item label="注册时间" v-if="auth.user?.created_at">
            <span>{{ formatDate(auth.user.created_at) }}</span>
          </el-form-item>
        </el-form>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { reactive, watch, ref, computed } from 'vue';
import { ElMessage } from 'element-plus';

import { useAuthStore } from '@/store/auth';
import { updateProfile } from '@/api/auth';

const auth = useAuthStore();
const loading = ref(false);
const form = reactive({
  name: '',
  email: '',
  phone: '',
});

// 生成头像URL或使用默认头像
const avatarUrl = computed(() => {
  // 这里可以根据用户邮箱生成Gravatar头像或使用其他头像服务
  // 如果有用户头像URL，可以直接返回
  return auth.user?.avatar || '';
});

// 格式化日期函数
const formatDate = (dateString) => {
  if (!dateString) return '';
  const date = new Date(dateString);
  return date.toLocaleString('zh-CN', { 
    year: 'numeric', 
    month: '2-digit', 
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
};

watch(
  () => auth.user,
  (user) => {
    form.name = user?.name || '';
    form.email = user?.email || '';
    form.phone = user?.phone || '';
  },
  { immediate: true },
);

const save = async () => {
  loading.value = true;
  try {
    const updated = await updateProfile({ name: form.name, phone: form.phone });
    auth.setSession({
      access_token: auth.token,
      refresh_token: auth.refreshToken,
      user: updated,
    });
    ElMessage.success('个人信息已更新');
  } catch (error) {
    console.error(error);
    ElMessage.error(error.response?.data?.message || '更新个人信息失败');
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.profile-container {
  padding: 32px;
}

.card {
  max-width: 800px;
  margin: 0 auto;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
  border-radius: 8px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-title {
  font-size: 18px;
  font-weight: 600;
}

.profile-content {
  display: flex;
  flex-direction: column;
  padding: 20px 0;
}

.avatar-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 30px;
}

.avatar {
  margin-bottom: 16px;
  background-color: #409EFF;
  color: white;
  font-size: 36px;
  font-weight: bold;
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.user-name {
  font-size: 20px;
  font-weight: 600;
  margin-top: 8px;
}

.form-section {
  max-width: 500px;
  margin: 0 auto;
  width: 100%;
}

@media (max-width: 768px) {
  .profile-container {
    padding: 16px;
  }
  
  .card {
    margin: 0;
  }
}
</style>
