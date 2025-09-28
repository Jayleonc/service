import ElementPlus from 'element-plus'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import 'element-plus/dist/index.css'

import pinia from '@/store'
import router from '@/router'
import { setupRouterGuards } from '@/router/guards'

export function registerPlugins(app) {
  app
    .use(pinia)
    .use(router)
    .use(ElementPlus, { locale: zhCn })

  setupRouterGuards(router)
}
