<template>
  <!-- 空布局：无导航栏 / 页脚，用于登录等独立页面 -->
  <div class="min-h-screen flex flex-col bg-surface-base text-fg-primary font-sans">
    <main class="flex-grow">
      <slot />
    </main>
  </div>
</template>
