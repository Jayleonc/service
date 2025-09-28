<template>
  <div class="view-container">
    <div class="toolbar">
      <el-input v-model="filters.name" placeholder="Search by name" clearable class="filter-input" @clear="loadUsers" @change="handleFilter" />
      <el-input v-model="filters.email" placeholder="Search by email" clearable class="filter-input" @clear="loadUsers" @change="handleFilter" />
      <el-button type="primary" @click="openCreate">Create User</el-button>
    </div>

    <el-table :data="users" border stripe>
      <el-table-column prop="name" label="Name" />
      <el-table-column prop="email" label="Email" />
      <el-table-column prop="phone" label="Phone" />
      <el-table-column label="Roles">
        <template #default="scope">
          <el-tag v-for="role in scope.row.roles" :key="role" class="role-tag">{{ role }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="Actions" width="220">
        <template #default="scope">
          <el-button size="small" @click="openEdit(scope.row)">Edit</el-button>
          <el-button size="small" type="warning" @click="openAssign(scope.row)">Assign Roles</el-button>
          <el-popconfirm title="Delete this user?" @confirm="removeUser(scope.row)">
            <template #reference>
              <el-button size="small" type="danger">Delete</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination">
      <el-pagination
        background
        layout="prev, pager, next"
        :total="pagination.total"
        :page-size="pagination.pageSize"
        :current-page="pagination.page"
        @current-change="handlePageChange"
      />
    </div>

    <el-dialog :title="dialog.title" v-model="dialog.visible" width="480px">
      <el-form :model="dialog.form" label-width="120px">
        <el-form-item label="Name">
          <el-input v-model="dialog.form.name" />
        </el-form-item>
        <el-form-item label="Email">
          <el-input v-model="dialog.form.email" :disabled="dialog.mode === 'edit'" />
        </el-form-item>
        <el-form-item label="Password" v-if="dialog.mode === 'create'">
          <el-input v-model="dialog.form.password" type="password" />
        </el-form-item>
        <el-form-item label="Phone">
          <el-input v-model="dialog.form.phone" />
        </el-form-item>
        <el-form-item label="Roles">
          <el-select v-model="dialog.form.roles" multiple placeholder="Select roles">
            <el-option v-for="role in roles" :key="role.id" :label="role.name" :value="role.name" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Cancel</el-button>
        <el-button type="primary" :loading="dialog.loading" @click="submitDialog">Save</el-button>
      </template>
    </el-dialog>

    <el-dialog title="Assign Roles" v-model="assignDialog.visible" width="420px">
      <el-form label-width="120px">
        <el-form-item label="User">
          <span>{{ assignDialog.user?.name }}</span>
        </el-form-item>
        <el-form-item label="Roles">
          <el-select v-model="assignDialog.roles" multiple placeholder="Select roles">
            <el-option v-for="role in roles" :key="role.id" :label="role.name" :value="role.name" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="assignDialog.visible = false">Cancel</el-button>
        <el-button type="primary" :loading="assignDialog.loading" @click="submitAssign">Assign</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { ElMessage } from 'element-plus';

import { listUsers, createUser, updateUser, deleteUser, assignUserRoles } from '@/api/users';
import { listRoles } from '@/api/roles';

const users = ref([]);
const roles = ref([]);
const filters = reactive({ name: '', email: '' });
const pagination = reactive({ page: 1, pageSize: 10, total: 0 });

const dialog = reactive({
  visible: false,
  mode: 'create',
  title: 'Create User',
  loading: false,
  form: {
    id: '',
    name: '',
    email: '',
    password: '',
    phone: '',
    roles: [],
  },
});

const assignDialog = reactive({
  visible: false,
  loading: false,
  user: null,
  roles: [],
});

const loadUsers = async () => {
  try {
    const result = await listUsers({ page: pagination.page, page_size: pagination.pageSize }, filters);
    users.value = result.list || [];
    pagination.total = result.total || 0;
    pagination.page = result.page || 1;
    pagination.pageSize = result.page_size || pagination.pageSize;
  } catch (error) {
    console.error(error);
    ElMessage.error('Failed to load users');
  }
};

const loadRoles = async () => {
  try {
    roles.value = await listRoles();
  } catch (error) {
    console.error(error);
    ElMessage.error('Failed to load roles');
  }
};

const handlePageChange = (page) => {
  pagination.page = page;
  loadUsers();
};

const handleFilter = () => {
  pagination.page = 1;
  loadUsers();
};

const resetDialog = () => {
  dialog.form = {
    id: '',
    name: '',
    email: '',
    password: '',
    phone: '',
    roles: [],
  };
};

const openCreate = () => {
  resetDialog();
  dialog.mode = 'create';
  dialog.title = 'Create User';
  dialog.visible = true;
};

const openEdit = (user) => {
  resetDialog();
  dialog.mode = 'edit';
  dialog.title = 'Edit User';
  dialog.form = {
    id: user.id,
    name: user.name,
    email: user.email,
    password: '',
    phone: user.phone,
    roles: [...(user.roles || [])],
  };
  dialog.visible = true;
};

const openAssign = (user) => {
  assignDialog.user = user;
  assignDialog.roles = [...(user.roles || [])];
  assignDialog.visible = true;
};

const submitDialog = async () => {
  dialog.loading = true;
  try {
    if (dialog.mode === 'create') {
      await createUser({
        name: dialog.form.name,
        email: dialog.form.email,
        password: dialog.form.password,
        phone: dialog.form.phone,
        roles: dialog.form.roles,
      });
      ElMessage.success('User created');
    } else {
      await updateUser({
        id: dialog.form.id,
        name: dialog.form.name,
        phone: dialog.form.phone,
      });
      if (dialog.form.roles?.length) {
        await assignUserRoles({ id: dialog.form.id, roles: dialog.form.roles });
      }
      ElMessage.success('User updated');
    }
    dialog.visible = false;
    loadUsers();
  } catch (error) {
    console.error(error);
    ElMessage.error(error.response?.data?.message || 'Failed to save user');
  } finally {
    dialog.loading = false;
  }
};

const submitAssign = async () => {
  assignDialog.loading = true;
  try {
    await assignUserRoles({ id: assignDialog.user.id, roles: assignDialog.roles });
    ElMessage.success('Roles updated');
    assignDialog.visible = false;
    loadUsers();
  } catch (error) {
    console.error(error);
    ElMessage.error(error.response?.data?.message || 'Failed to assign roles');
  } finally {
    assignDialog.loading = false;
  }
};

const removeUser = async (user) => {
  try {
    await deleteUser(user.id);
    ElMessage.success('User deleted');
    loadUsers();
  } catch (error) {
    console.error(error);
    ElMessage.error(error.response?.data?.message || 'Failed to delete user');
  }
};

onMounted(() => {
  loadUsers();
  loadRoles();
});
</script>

<style scoped>
.view-container {
  background: #fff;
  padding: 24px;
  border-radius: 8px;
}

.toolbar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  align-items: center;
}

.filter-input {
  width: 220px;
}

.pagination {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}

.role-tag {
  margin-right: 4px;
}
</style>
