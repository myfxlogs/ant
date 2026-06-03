import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    redirect: '/strategy/workspace',
  },
  {
    path: '/strategy/workspace',
    name: 'StrategyWorkspace',
    component: () => import('@/views/strategy/StrategyWorkspace.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

export default router
