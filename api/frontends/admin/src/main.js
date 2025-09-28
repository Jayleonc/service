import { createApp } from 'vue'
import App from './App.vue'
import { registerPlugins } from '@/plugins'
import { useAuthStore } from '@/store/auth'
import '@/styles/main.css'

const app = createApp(App)

registerPlugins(app)

const auth = useAuthStore()
auth.hydrate()

app.mount('#app')
