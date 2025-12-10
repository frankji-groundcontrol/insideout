# 卷了么 (JuanLeMe) 开发实施计划

基于 `INIT.md` 产品定义与 `project_rules.md` 规范，制定本全栈开发计划。

## 1. 开发策略调整：纯前端先行 (Frontend First Strategy)

**核心策略**：优先构建高保真、可交互的纯前端应用，使用 Mock Data (模拟数据) 替代真实后端，专注于验证用户交互与视觉效果。待前端逻辑与体验完全确认后，再进行后端对接。

### 阶段划分
1.  **阶段一：静态交互验证 (Mock Mode)**
    *   构建所有 UI 界面与路由跳转。
    *   使用本地 JSON/TS 文件模拟数据库返回。
    *   实现除真实数据存储外的所有交互逻辑 (表单校验、状态切换、富文本编辑、AI对话模拟)。
    *   **目标**: 用户可以完整体验“注册 -> 任务 -> 导出”的全流程，感觉像是在用真实 App。

2.  **阶段二：后端对接与联调 (Integration Mode)**
    *   搭建 Supabase 数据库与 API。
    *   替换 Mock Service 为真实 Supabase Client。
    *   实现真实认证与数据持久化。

---

## 2. 技术选型细节 (Technology Stack)

### 前端架构 (Frontend)
*   **核心框架**: Vue 3.4+ (使用 `<script setup>` 语法)
*   **构建工具**: Vite 5.x
*   **状态管理**: Pinia 2.x (用于管理用户信息、工作坊状态、UI状态)
*   **路由管理**: Vue Router 4.x
*   **UI 样式**: TailwindCSS 3.x (优先使用 Utility classes 实现响应式) + Headless UI (无样式组件，保证可定制性)
*   **富文本编辑器**: Tiptap (Vue 3 适配好，扩展性强，支持 Markdown)
*   **数据模拟**: 
    *   使用 TypeScript 定义标准接口 (`interfaces`)。
    *   创建 `services/mock` 目录，提供与真实 API 结构一致的模拟数据。
*   **HTTP 客户端**: Supabase JS Client (阶段二引入)
*   **Markdown 渲染**: markdown-it

### 后端架构 (Backend - Supabase) - *阶段二实施*
*   **数据库**: PostgreSQL 15+
*   **认证 (Auth)**: Supabase Auth (Email/Password)
*   **API**: PostgREST (自动生成的 RESTful API)
*   **安全性**: RLS (Row Level Security) 严格控制数据访问
*   **存储**: Supabase Storage (用于存储用户头像、工作坊封面、上传的图片)
*   **AI 集成**: Edge Functions (调用 OpenAI/DeepSeek 接口)

---

## 2. 界面设计与交互规范 (UI/UX Design)

### 全局设计规范
*   **响应式策略**: Mobile First (优先设计移动端)，通过 Tailwind `md:`, `lg:` 适配平板和桌面。
*   **主题**: 支持亮色/暗色模式 (Dark Mode)，适配系统设置。
*   **交互反馈**: 按钮点击有 `active` 态，耗时操作显示 Loading Spinner，成功/失败操作显示 Toast 提示。

### 核心界面与交互流程

#### A. 公共模块 (Public)

1.  **落地页/登录注册页 (Landing/Auth)**
    *   **功能**: 展示产品Slogan，提供登录/注册入口。
    *   **路径**: 访客 -> 注册(邮箱/密码/用户名) -> 自动登录 -> 首页。
    *   **UI元素**: 简洁的表单卡片，品牌Logo，"开始旅程"按钮。

#### B. 首页/工作坊列表 (Dashboard)

1.  **工作坊列表页 (Workshop List)**
    *   **功能**: 展示“我参与的”和“我管理的”工作坊。
    *   **角色差异**:
        *   **所有用户**: 看到列表，右上角有“加入工作坊”按钮(输入邀请码)。
        *   **组织者**: 额外拥有“创建工作坊”按钮。
    *   **交互**: 点击卡片 -> 进入工作坊详情(Roadmap)。
    *   **UI元素**: 卡片式布局（封面、标题、角色标签、状态），悬浮操作按钮(FAB)在移动端用于新建/加入。

#### C. 工作坊详情 (Workshop Detail)

1.  **路线图页 (Roadmap View)**
    *   **功能**: 可视化展示活动流程。
    *   **交互**:
        *   **参与者**: 查看流程节点，点击“进行中”或“未开始”的节点进入任务。
        *   **组织者**: 点击“编辑模式”可拖拽排序、添加节点、编辑节点内容。
    *   **UI元素**: 垂直时间轴或卡片流，节点状态颜色区分（灰:未开始, 蓝:进行中, 绿:已完成）。

#### D. 任务执行页 (Task Execution / Editor)

1.  **富文本/AI 协作页**
    *   **功能**: 核心页面。左侧/上方为题目要求，中部为富文本编辑区，右侧/悬浮为 AI Chatbot。
    *   **交互**:
        *   用户输入内容 -> AI 侧边栏提供灵感/润色建议 -> 一键插入编辑器。
        *   支持 Markdown 快捷键。
        *   实时保存草稿。
    *   **UI元素**: Tiptap 工具栏，分栏布局（桌面端）/ Tab切换布局（移动端）。

2.  **结果导出页**
    *   **功能**: 将所有任务回答汇总。
    *   **交互**: 预览 -> 选择格式 (PDF/MD) -> 下载。

#### E. 个人中心 (Profile)

1.  **个人资料页**
    *   **功能**: 编辑昵称、头像、简介、关键词。
    *   **交互**: 点击头像上传（调用系统相册/文件）。

---

## 3. 文件结构规划 (File Structure)

遵循模块化、单一职责原则，单文件 < 500行。

```text
src/
├── assets/                 # 静态资源 (Logo, Icons)
├── components/             # 通用组件 (Base UI)
│   ├── common/             # 按钮, 输入框, 弹窗
│   │   ├── BaseButton.vue
│   │   ├── BaseInput.vue
│   │   └── ModalDialog.vue
│   ├── layout/             # 布局组件
│   │   ├── NavBar.vue
│   │   └── AppFooter.vue
│   └── workshop/           # 业务组件
│       ├── WorkshopCard.vue
│       └── RoadmapItem.vue
├── views/                  # 页面视图 (Pages)
│   ├── auth/
│   │   ├── LoginPage.vue
│   │   └── RegisterPage.vue
│   ├── dashboard/
│   │   └── WorkshopList.vue
│   ├── workshop/
│   │   ├── WorkshopDetail.vue
│   │   └── TaskEditor.vue
│   └── profile/
│       └── UserProfile.vue
├── stores/                 # Pinia 状态管理
│   ├── user.ts             # 用户认证与个人信息
│   ├── workshop.ts         # 工作坊列表与当前选中状态
│   └── editor.ts           # 编辑器内容与自动保存
├── services/               # API 服务层 (Supabase)
│   ├── supabase.ts         # 初始化客户端
│   ├── authService.ts
│   ├── dataService.ts
│   └── aiService.ts        # 调用 Edge Functions
├── utils/                  # 工具函数
│   ├── format.ts           # 日期/文本格式化
│   └── export.ts           # 导出 PDF/MD 逻辑
├── types/                  # TypeScript 类型定义
│   └── supabase.ts         # 数据库类型 (自动生成)
├── App.vue
└── main.ts
```

---

## 4. 数据库设计 (Database Schema)

*所有表必须启用 RLS (Row Level Security)。*

1.  **profiles** (用户表，扩展 auth.users)
    *   `id`: uuid (PK, ref auth.users)
    *   `username`: text
    *   `avatar_url`: text
    *   `bio`: text
    *   `keywords`: text[]
    *   `role`: text ('user' | 'admin') -- 系统级角色，默认 user

2.  **workshops** (工作坊)
    *   `id`: uuid (PK)
    *   `name`: text
    *   `description`: text
    *   `invite_code`: text (Unique)
    *   `owner_id`: uuid (FK profiles.id)
    *   `created_at`: timestamptz

3.  **workshop_members** (成员关联表)
    *   `workshop_id`: uuid (FK)
    *   `user_id`: uuid (FK)
    *   `role`: text ('participant' | 'organizer') -- 工作坊级角色
    *   `joined_at`: timestamptz
    *   *PK: (workshop_id, user_id)*

4.  **roadmap_nodes** (流程节点)
    *   `id`: uuid (PK)
    *   `workshop_id`: uuid (FK)
    *   `title`: text
    *   `description`: text (支持 Markdown)
    *   `order_index`: integer (排序)
    *   `node_type`: text ('reading' | 'question' | 'input')

5.  **user_submissions** (用户回答)
    *   `id`: uuid (PK)
    *   `node_id`: uuid (FK)
    *   `user_id`: uuid (FK)
    *   `content`: text (用户填写的富文本内容)
    *   `status`: text ('draft' | 'submitted')
    *   `updated_at`: timestamptz

---

## 5. 兼容性实现方案 (Compatibility)

*   **断点设置 (Tailwind)**:
    *   `DEFAULT`: 移动端 (手机竖屏) - 100% 宽度，单栏布局。
    *   `md` (768px): 平板 (iPad) - 侧边栏出现，双栏布局。
    *   `lg` (1024px): 桌面端 - 居中容器，最大宽度限制。

*   **触摸优化**:
    *   所有可点击区域 (Button, Icon) 最小尺寸 44x44px。
    *   避免 Hover 交互作为唯一触发方式 (移动端无 Hover)。
    *   编辑器工具栏在移动端需适配键盘弹出，避免遮挡。

*   **测试设备清单**:
    *   iPhone (Safari/Chrome)
    *   Android Phone (Chrome)
    *   iPad (Safari)
    *   Mac (Chrome/Safari)
    *   Windows PC (Edge/Chrome)

---

## 6. 代码注释规范 (Comments)

**必须严格执行 `project_rules.md` 中的注释要求。**

*   **文件头**: 描述文件功能。
*   **组件**: 使用 JSDoc 格式描述 `Props`, `Emits`, `Slots`。
*   **复杂逻辑**: 必须分步解释。
*   **语言**: **全中文**。

示例：
```typescript
/**
 * @function joinWorkshop
 * @description 用户通过邀请码加入工作坊
 * @param {string} code - 6位邀请码
 * @returns {Promise<boolean>} - 加入成功返回 true
 * @logic 
 * 1. 验证邀请码格式
 * 2. 调用 Supabase RPC 函数验证并插入成员记录
 * 3. 失败则抛出具体错误信息给前端展示
 */
```

---

## 8. 开发进度与里程碑 (Milestones)

### 阶段一：纯前端交互开发 (Mock Mode) - *当前重点*
*   **Step 1: 基础设施搭建**
    *   初始化 Vue3 + Vite + Tailwind + Pinia 项目。
    *   搭建基础 Layout (NavBar, Footer) 和路由结构。
    *   定义核心 TypeScript 类型 (`User`, `Workshop`, `Node`)。
*   **Step 2: 核心页面 UI 实现**
    *   实现 Dashboard (工作坊列表) 静态页。
    *   实现 Roadmap (路线图) 静态交互页。
    *   实现 Editor (富文本编辑器) 集成 Tiptap。
*   **Step 3: 交互逻辑串联 (Mock Data)**
    *   创建 `services/mock` 服务，提供模拟数据。
    *   实现“登录 -> 列表 -> 详情 -> 答题”的完整前端跳转逻辑。
    *   模拟 AI 侧边栏对话效果（硬编码或随机回复）。
    *   **交付物**: 一个可以在浏览器中完整跑通流程的“单机版”应用，用户体验与最终产品一致。

### 阶段二：后端集成与上线 (Integration Mode) - *后续规划*
*   **Step 4: Supabase 接入**
    *   配置数据库表结构与 RLS。
    *   替换 Mock Service 为 Real Service。
    *   对接真实 AI Edge Functions。
*   **Step 5: 最终验收**
    *   全链路测试。
    *   多设备兼容性修复。

---

## 9. 测试与验收标准

1.  **交互验证 (阶段一)**: 在 Mock 模式下，用户能流畅完成所有操作，无 JS 报错，动画流畅。
2.  **视觉还原**: 100% 还原 UI 设计规范，响应式布局在手机/PC 端均正常。
3.  **兼容性**: 覆盖主流浏览器与设备。
