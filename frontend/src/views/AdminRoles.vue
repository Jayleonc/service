<template>
  <div class="view-container">
    <div class="toolbar">
      <el-button type="primary" @click="openCreate">Create Role</el-button>
    </div>

    <el-table :data="roles" border stripe>
      <el-table-column prop="name" label="Name" />
      <el-table-column prop="description" label="Description" />
      <el-table-column prop="createdAt" label="Created At" />
      <el-table-column label="Actions" width="180">
        <template #default="scope">
          <el-button size="small" @click="openEdit(scope.row)">Edit</el-button>
          <el-popconfirm title="Delete this role?" @confirm="removeRole(scope.row)">
            <template #reference>
              <el-button size="small" type="danger">Delete</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog :title="dialog.title" v-model="dialog.visible" width="420px">
      <el-form :model="dialog.form" label-width="120px">
        <el-form-item label="Name">
          <el-input v-model="dialog.form.name" :disabled="dialog.mode === 'edit'" />
        </el-form-item>
        <el-form-item label="Description">
          <el-input v-model="dialog.form.description" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">Cancel</el-button>
        <el-button type="primary" :loading="dialog.loading" @click="submitDialog">Save</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue';
import { ElMessage } from 'element-plus';

import { listRoles, createRole, updateRole, deleteRole } from '@/api/roles';

const roles = ref([]);

const dialog = reactive({
  visible: false,
  loading: false,
  mode: 'create',
  title: 'Create Role',
  form: {
    id: '',
    name: '',
    description: '',
  },
});

const loadRoles = async () => {
  try {
    roles.value = await listRoles();
  } catch (error) {
    console.error(error);
    ElMessage.error('Failed to load roles');
  }
};

const resetDialog = () => {
  dialog.form = { id: '', name: '', description: '' };
};

const openCreate = () => {
  resetDialog();
  dialog.mode = 'create';
  dialog.title = 'Create Role';
  dialog.visible = true;
};

const openEdit = (role) => {
  resetDialog();
  dialog.mode = 'edit';
  dialog.title = 'Edit Role';
  dialog.form = { id: role.id, name: role.name, description: role.description };
  dialog.visible = true;
};

const submitDialog = async () => {
  dialog.loading = true;
  try {
    if (dialog.mode === 'create') {
      await createRole({ name: dialog.form.name, description: dialog.form.description });
      ElMessage.success('Role created');
    } else {
      await updateRole({ id: dialog.form.id, name: dialog.form.name, description: dialog.form.description });
      ElMessage.success('Role updated');
    }
    dialog.visible = false;
    loadRoles();
  } catch (error) {
    console.error(error);
    ElMessage.error(error.response?.data?.message || 'Failed to save role');
  } finally {
    dialog.loading = false;
  }
};

const removeRole = async (role) => {
  try {
    await deleteRole(role.id);
    ElMessage.success('Role deleted');
    loadRoles();
  } catch (error) {
    console.error(error);
    ElMessage.error(error.response?.data?.message || 'Failed to delete role');
  }
};

onMounted(() => {
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
  justify-content: flex-end;
  margin-bottom: 16px;
}
</style>
