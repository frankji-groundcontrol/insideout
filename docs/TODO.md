# 卷了么 (JuanLeMe) 开发进度清单

> 本文档用于实时追踪项目开发进度。每完成一项，请将 `[ ]` 标记为 `[x]`，并填写负责人和完成日期。
> **当前开发模式**: 纯前端 Mock 模式 (Frontend First)

---

## � 项目文件结构 (File Structure)

> ⚠️ **注意**: 每次新增重要文件或模块时，请同步更新此结构树。

```text
src/
├── assets/                 # 静态资源 (Logo, Icons)
├── components/             # 通用组件 (Base UI)
│   ├── common/             # [ ] 基础组件 (Button, Input, Modal)
│   ├── layout/             # [ ] 布局组件 (NavBar, Footer)
│   └── workshop/           # [ ] 业务组件 (Card, RoadmapItem, AiSidebar)
├── views/                  # 页面视图 (Pages)
│   ├── auth/               # [ ] 认证页 (Login, Register)
│   ├── dashboard/          # [ ] 列表页 (WorkshopList)
│   ├── workshop/           # [ ] 详情页 (Detail, Editor, Export)
│   └── profile/            # [ ] 个人页 (UserProfile)
├── stores/                 # Pinia 状态管理
│   ├── user.ts             # [ ] 用户状态
│   ├── workshop.ts         # [ ] 工作坊状态
│   └── editor.ts           # [ ] 编辑器状态
├── services/               # API 服务层
│   ├── mock/               # [ ] Mock 数据与接口
│   └── supabase/           # [ ] (阶段二) 真实 API
├── utils/                  # 工具函数
├── types/                  # TypeScript 类型定义
├── App.vue                 # [ ] 根组件
└── main.ts                 # [ ] 入口文件
```

---

## �📅 阶段一：纯前端交互开发 (Mock Mode)

### 🚀 Step 1: 基础设施搭建 (Infrastructure)

- [x] **1.1 项目初始化**
    - [x] 使用 Vite 创建 Vue 3 + TypeScript 项目框架
    - [x] 配置 TailwindCSS (安装依赖, `tailwind.config.js`, `index.css`)
    - [x] 安装并配置 Pinia (状态管理)
    - [x] 安装并配置 Vue Router 4 (路由管理)
    - [x] 配置 `tsconfig.json` 路径别名 (`@/`)
    - [x] 配置 ESLint + Prettier 代码规范
    - [x] 提交初次 Commit
    - *负责人: Trae | 完成日期: 2025-12-24*

- [x] **1.2 基础布局与组件 (Layout & Base UI)**
    - [x] 创建全局布局组件 `src/App.vue` (RouterView)
    - [x] 实现导航栏组件 `src/components/layout/NavBar.vue` (响应式, 支持暗色模式切换)
    - [x] 实现底部组件 `src/components/layout/AppFooter.vue`
    - [x] 封装基础按钮 `src/components/common/BaseButton.vue` (多种 variant: primary, secondary, outline)
    - [x] 封装基础输入框 `src/components/common/BaseInput.vue` (支持 label, error message)
    - *负责人: Trae | 完成日期: 2025-12-24*

- [ ] **1.3 类型定义与 Mock 数据 (Types & Mocks)**
    - [x] 定义核心类型 `src/types/user.ts` (UserProfile)
    - [x] 定义核心类型 `src/types/workshop.ts` (Workshop, RoadmapNode)
    - [x] 创建 Mock 服务 `src/services/mock/mockData.ts` (包含 3 个预设工作坊, 5 个节点)
    - [x] 封装 Mock API 接口 `src/services/mock/index.ts` (模拟异步请求 delay)
    - *负责人: Trae | 完成日期: 2025-12-24*

### 🎨 Step 2: 核心页面 UI 实现 (Core Views)

- [ ] **2.1 认证模块 (Auth Views)**
    - [ ] 实现登录页 `src/views/auth/LoginPage.vue` (静态 UI)
    - [ ] 实现注册页 `src/views/auth/RegisterPage.vue` (静态 UI)
    - [ ] 实现 Pinia `userStore` (模拟登录/注册/退出逻辑)
    - *负责人: 待定 | 完成日期: -*

- [ ] **2.2 工作坊列表 (Dashboard)**
    - [x] 创建工作坊列表页 `src/views/dashboard/DashboardView.vue` (卡片式布局)
    - [x] 封装工作坊卡片组件 `src/components/workshop/WorkshopCard.vue`
    - [x] 实现列表响应式适配 (auto-fit grid layout)
    - [x] 实现工作坊详情页 `src/views/workshop/WorkshopDetailView.vue` (左右分栏布局)
    - [x] 封装路线图侧边栏 `src/components/workshop/RoadmapSidebar.vue`
    - [x] 封装任务详情组件 `src/components/workshop/TaskDetail.vue`
    - *负责人: Trae | 完成日期: 2025-12-24*

- [ ] **2.3 个人中心 (Profile)**
    - [ ] 实现个人资料页 `src/views/profile/UserProfile.vue`
    - [ ] 实现头像上传 (Mock: 仅本地预览) 与表单编辑
    - *负责人: 待定 | 完成日期: -*

### 🧩 Step 3: 流程与内容系统 (Roadmap & Editor)

- [ ] **3.1 路线图页 (Roadmap View)**
    - [ ] 实现路线图节点组件 `src/components/workshop/RoadmapItem.vue`
    - [ ] 实现路线图详情页 `src/views/workshop/WorkshopDetail.vue`
    - [ ] 实现节点状态逻辑 (未开始/进行中/已完成 样式区分)
    - [ ] 实现 Pinia `workshopStore` (管理当前工作坊数据)
    - *负责人: 待定 | 完成日期: -*

- [ ] **3.2 富文本编辑器 (Task Editor)**
    - [ ] 集成 Tiptap 编辑器 (`npm install @tiptap/vue-3 ...`)
    - [ ] 实现编辑器组件 `src/views/workshop/TaskEditor.vue`
    - [ ] 自定义编辑器工具栏 (加粗, 列表, 标题)
    - [ ] 实现 Markdown 快捷键支持
    - [ ] 实现 Pinia `editorStore` (自动保存草稿到 localStorage)
    - *负责人: 待定 | 完成日期: -*

- [ ] **3.3 AI 协作侧边栏 (AI Chatbot)**
    - [ ] 实现侧边栏组件 `src/components/workshop/AiSidebar.vue`
    - [ ] 模拟 AI 对话逻辑 (发送消息 -> 随机延迟 -> 返回预设回复)
    - [ ] 实现“一键采纳”功能 (将 AI 回复插入编辑器)
    - *负责人: 待定 | 完成日期: -*

- [ ] **3.4 导出功能 (Export)**
    - [ ] 实现导出预览页 `src/views/workshop/ExportPreview.vue`
    - [ ] 实现 Markdown/PDF 导出逻辑 (使用 `html2canvas` + `jspdf` 或纯文本生成)
    - *负责人: 待定 | 完成日期: -*

---

## ⏳ 阶段二：后端集成与上线 (Integration Mode) - *待启动*

### 🔌 Step 4: Supabase 接入
- [ ] 数据库表结构迁移 (Migrations)
- [ ] 配置 RLS (Row Level Security) 策略
- [ ] 替换 Mock Service 为 Real Service (`src/services/supabase/`)
- [ ] 对接真实 AI Edge Functions

### ✅ Step 5: 最终验收
- [ ] 全链路功能测试
- [ ] 多设备兼容性测试 (Mobile/Tablet/Desktop)
- [ ] Bug 修复与性能优化

---

## 📝 变更日志 (Changelog)

- **2025-12-10**: 初始化 `TODO.md`，确立纯前端开发阶段 (Mock Mode) 的详细任务清单。
