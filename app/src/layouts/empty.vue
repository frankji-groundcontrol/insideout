<template>
  <!-- 空布局：无导航栏 / 页脚，用于登录等独立页面 -->
  <div class="min-h-screen flex flex-col bg-gray-50 text-gray-900 font-sans dark:bg-gray-900 dark:text-gray-100">
    <main class="flex-grow">
      <slot />
    </main>
  </div>
</template>
