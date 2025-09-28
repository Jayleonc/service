/**
 * main.js
 *
 * Bootstraps Vuetify and other plugins then mounts the App`
 */

// Components
import App from './App.vue'

// Composables
import { createApp } from 'vue'

// Plugins
import { registerPlugins } from '@/plugins'
import { useAuthStore } from '@/store/auth'

const app = createApp(App)

registerPlugins(app)

// hydrate auth state before mount
const auth = useAuthStore()
auth.hydrate()

app.mount('#app')
