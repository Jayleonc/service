<template>
  <el-container class="admin-layout">
    <el-aside width="220px" class="sidebar">
      <div class="brand">Service Admin</div>
      <el-menu :default-active="activeMenu" class="menu" @select="handleSelect">
        <el-menu-item index="/admin/users">User Management</el-menu-item>
        <el-menu-item index="/admin/roles">Role Management</el-menu-item>
        <el-menu-item index="/me">My Profile</el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <div class="spacer" />
        <div class="user-info">
          <span class="name">{{ auth.user?.name || 'Unknown' }}</span>
          <el-tag v-for="role in auth.roles" :key="role" class="role-tag" type="info">{{ role }}</el-tag>
          <el-button type="text" @click="handleLogout">Logout</el-button>
        </div>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessageBox } from 'element-plus';

import { useAuthStore } from '@/store/auth';

const router = useRouter();
const route = useRoute();
const auth = useAuthStore();

const activeMenu = computed(() => {
  if (route.path.startsWith('/admin/')) {
    return `/admin/${route.path.split('/')[2]}`;
  }
  return route.path;
});

const handleSelect = (path) => {
  router.push(path);
};

const handleLogout = async () => {
  const confirmed = await ElMessageBox.confirm(
    'Are you sure you want to logout?',
    'Confirm Logout',
    {
      confirmButtonText: 'Logout',
      cancelButtonText: 'Cancel',
      type: 'warning',
    },
  ).catch(() => false);

  if (confirmed) {
    auth.logout();
    router.replace('/login');
  }
};
</script>

<style scoped>
.admin-layout {
  min-height: 100vh;
}

.sidebar {
  background-color: #1f2a44;
  color: #fff;
  display: flex;
  flex-direction: column;
}

.brand {
  font-size: 20px;
  font-weight: 600;
  padding: 24px 16px;
  text-align: center;
}

.menu {
  border-right: none;
  flex: 1;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background-color: #fff;
  border-bottom: 1px solid #e4e7ed;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.name {
  font-weight: 600;
}

.role-tag {
  text-transform: capitalize;
}
</style>
