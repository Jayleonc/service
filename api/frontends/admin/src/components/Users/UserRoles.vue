<template>
  <v-card class="mt-4">
    <v-card-title class="d-flex align-center justify-space-between">
      <div>角色管理</div>
      <div class="d-flex align-center">
        <v-text-field v-model="newRole" density="compact" label="新增角色" hide-details class="mr-2" @keyup.enter="addRole" />
        <v-btn size="small" color="primary" @click="addRole">添加</v-btn>
      </div>
    </v-card-title>
    <v-card-text>
      <div class="mb-4">
        <v-chip
          v-for="r in roles"
          :key="r"
          class="ma-1"
          closable
          @click:close="removeRole(r)"
        >{{ r }}</v-chip>
        <div v-if="!roles.length" class="text-medium-emphasis">暂无角色</div>
      </div>
      <v-alert type="error" v-if="error" class="mb-2">{{ error }}</v-alert>
      <v-btn :loading="saving" color="primary" @click="save">保存变更</v-btn>
    </v-card-text>
  </v-card>
</template>

<script>
import { getUserRoles, setUserRoles, addUserRole } from '@/api/roles'

export default {
  name: 'UserRoles',
  props: {
    userId: { type: String, default: '' },
  },
  data () {
    return {
      roles: [],
      newRole: '',
      saving: false,
      error: null,
    }
  },
  watch: {
    userId: {
      immediate: true,
      handler (val) {
        if (val) this.load()
      },
    },
  },
  methods: {
    async load () {
      this.error = null
      try {
        const data = await getUserRoles(this.userId)
        this.roles = Array.isArray(data.roles) ? data.roles : []
      } catch (e) {
        this.error = e.message || '加载角色失败'
      }
    },
    addRole () {
      const r = (this.newRole || '').trim().toLowerCase()
      if (!r) return
      if (!this.roles.includes(r)) this.roles.push(r)
      this.newRole = ''
    },
    removeRole (r) {
      this.roles = this.roles.filter(x => x !== r)
    },
    async save () {
      this.saving = true
      this.error = null
      try {
        await setUserRoles(this.userId, this.roles)
        await this.load()
      } catch (e) {
        this.error = e.message || '保存失败'
      } finally {
        this.saving = false
      }
    },
    // 备用：单独添加一条（后端提供 POST /v1/roles）
    async addOne () {
      const r = (this.newRole || '').trim().toLowerCase()
      if (!r) return
      try {
        await addUserRole(this.userId, r)
        await this.load()
        this.newRole = ''
      } catch (e) {
        this.error = e.message || '添加失败'
      }
    }
  }
}
</script>
