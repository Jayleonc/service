<template>
  <el-card class="card">
    <template #header>
      <div class="card-header">
        <span>My Profile</span>
        <el-button type="primary" :loading="loading" @click="save">Save</el-button>
      </div>
    </template>
    <el-form :model="form" label-width="120px">
      <el-form-item label="Name">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="Email">
        <el-input v-model="form.email" disabled />
      </el-form-item>
      <el-form-item label="Phone">
        <el-input v-model="form.phone" />
      </el-form-item>
      <el-form-item label="Roles">
        <el-tag v-for="role in auth.roles" :key="role" type="success" class="role-tag">{{ role }}</el-tag>
      </el-form-item>
    </el-form>
  </el-card>
</template>

<script setup>
import { reactive, watch, ref } from 'vue';
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
    ElMessage.success('Profile updated');
  } catch (error) {
    console.error(error);
    ElMessage.error(error.response?.data?.message || 'Failed to update profile');
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.card {
  max-width: 640px;
  margin: 32px auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.role-tag {
  margin-right: 8px;
}
</style>
