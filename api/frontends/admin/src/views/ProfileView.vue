<template>
  <div class="profile">
    <el-card class="profile__card" shadow="hover">
      <div class="profile__header">
        <div>
          <h2 class="profile__title">个人信息</h2>
          <p class="profile__subtitle">查看并更新账户资料</p>
        </div>
        <el-tag type="info" v-if="user?.roles?.length">{{ user.roles.join(', ') }}</el-tag>
      </div>
      <el-divider />
      <el-skeleton :loading="loading" animated :rows="4">
        <el-form ref="formRef" :model="form" :rules="rules" label-width="100px" @submit.prevent="save">
          <el-form-item label="用户ID">
            <el-input :model-value="user?.id" readonly />
          </el-form-item>
          <el-form-item label="邮箱">
            <el-input :model-value="user?.email" readonly />
          </el-form-item>
          <el-form-item label="姓名" prop="name">
            <el-input v-model="form.name" placeholder="请输入姓名" />
          </el-form-item>
          <el-form-item label="手机号" prop="phone">
            <el-input v-model="form.phone" placeholder="请输入手机号" />
          </el-form-item>
          <el-form-item>
            <el-space>
              <el-button type="primary" :loading="saving" native-type="submit">保存</el-button>
              <el-button @click="reset">重置</el-button>
            </el-space>
          </el-form-item>
        </el-form>
      </el-skeleton>
    </el-card>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/store/auth'

const store = useAuthStore()
const formRef = ref(null)
const loading = ref(false)
const saving = ref(false)

const form = reactive({
  name: '',
  phone: '',
})

const rules = {
  name: [{ required: true, message: '姓名不能为空', trigger: 'blur' }],
  phone: [{ max: 20, message: '手机号太长', trigger: 'blur' }],
}

const user = computed(() => store.user)

function syncForm() {
  form.name = store.user?.name ?? ''
  form.phone = store.user?.phone ?? ''
}

async function load() {
  loading.value = true
  try {
    await store.fetchMe()
    syncForm()
  } catch (err) {
    ElMessage.error(err?.message || '获取个人信息失败')
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await store.updateMe({ name: form.name, phone: form.phone })
    ElMessage.success('保存成功')
    syncForm()
  } catch (err) {
    ElMessage.error(err?.message || '保存失败，请稍后再试')
  } finally {
    saving.value = false
  }
}

function reset() {
  syncForm()
}

onMounted(() => {
  load()
})
</script>

<style scoped>
.profile {
  width: min(720px, 100%);
}

.profile__card {
  border-radius: 18px;
}

.profile__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.profile__title {
  margin: 0;
  font-size: 26px;
  font-weight: 600;
  color: #1f2937;
}

.profile__subtitle {
  margin: 4px 0 0;
  color: #6b7280;
}
</style>
