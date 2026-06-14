<script setup lang="ts">
// 首页（公开）：渲染组件展示厅 HelloWorld。
// 路由元信息 public:true 供 middleware/auth.global.ts 放行未认证访问。
definePageMeta({
  public: true,
})
</script>

<template>
  <HelloWorld />
</template>
