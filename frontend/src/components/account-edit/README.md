# 邮箱账户智能编辑功能

## 概述

这个模块实现了智能的邮箱账户编辑功能，根据不同的邮箱类型（provider）和认证方式（auth_method）动态显示不同的可编辑字段，复用了添加邮箱功能的组件和验证逻辑。

## 功能特点

### 1. 智能类型识别
- 根据 `account.provider` 和 `account.auth_method` 自动识别邮箱类型
- 支持 Gmail OAuth2、Gmail 应用专用密码、Outlook OAuth2、Outlook 手动配置、QQ邮箱、163邮箱、自定义邮箱等

### 2. 差异化编辑界面
- **OAuth2 类型**：主要编辑账户名称，提供重新授权功能
- **密码类型**：可编辑账户名称、邮箱地址、密码/授权码
- **自定义类型**：可编辑完整的 IMAP/SMTP 配置

### 3. 复用现有组件
- 复用添加邮箱表单的字段组件和验证逻辑
- 保持一致的用户体验和代码质量

## 文件结构

```
account-edit/
├── account-edit-form.tsx          # 主入口组件，根据类型路由到不同表单
├── account-edit-config.ts         # 配置文件，定义不同类型的编辑规则
├── basic-edit-form.tsx            # 基础编辑表单（密码类型邮箱）
├── oauth2-edit-form.tsx           # OAuth2 编辑表单
├── custom-edit-form.tsx           # 自定义邮箱编辑表单
└── README.md                      # 说明文档
```

## 组件说明

### AccountEditForm
主入口组件，根据邮箱类型路由到对应的编辑表单。

### AccountEditConfig
配置文件，定义了不同邮箱类型的编辑规则：
- `type`: 表单类型（oauth2、basic、custom）
- `editableFields`: 可编辑的字段列表
- `showReauth`: 是否显示重新授权按钮
- `showPassword`: 是否显示密码字段
- `showImapSmtp`: 是否显示 IMAP/SMTP 配置
- `showOAuth2Config`: 是否显示 OAuth2 配置

### BasicEditForm
用于密码类型的邮箱（Gmail 应用专用密码、QQ邮箱、163邮箱等）：
- 可编辑账户名称、邮箱地址、密码/授权码
- 根据邮箱类型显示不同的密码字段标签

### OAuth2EditForm
用于 OAuth2 类型的邮箱：
- 主要编辑账户名称和启用状态
- 提供重新授权功能
- 支持手动 OAuth2 配置的编辑

### CustomEditForm
用于自定义邮箱：
- 可编辑完整的 IMAP/SMTP 配置
- 支持用户名、密码等认证信息
- 提供完整的服务器配置选项

## 支持的邮箱类型

| 类型 | Provider | Auth Method | 可编辑字段 | 特殊功能 |
|------|----------|-------------|------------|----------|
| Gmail OAuth2 | gmail | oauth2 | name, is_active | 重新授权 |
| Gmail 应用专用密码 | gmail | password | name, email, password, is_active | - |
| Outlook OAuth2 | outlook | oauth2 | name, is_active | 重新授权 |
| Outlook 手动配置 | outlook | oauth2_manual | name, email, client_id, client_secret, refresh_token, is_active | OAuth2 配置 |
| QQ邮箱 | qq | password | name, email, password, is_active | - |
| 163邮箱 | 163/netease | password | name, email, password, is_active | - |
| 自定义邮箱 | custom | password | 所有字段 | 完整配置 |

## 使用方法

```tsx
import { AccountEditForm } from '@/components/account-edit/account-edit-form';

function MyComponent() {
  const handleSuccess = () => {
    // 处理成功回调
  };

  const handleCancel = () => {
    // 处理取消回调
  };

  const updateAccount = (account: EmailAccount) => {
    // 更新账户数据
  };

  return (
    <AccountEditForm
      account={account}
      onSuccess={handleSuccess}
      onCancel={handleCancel}
      updateAccount={updateAccount}
    />
  );
}
```

## 测试页面

访问 `/test-edit` 页面可以测试不同类型邮箱的编辑功能。

## 设计原则

1. **SOLID 原则**：每个组件职责单一，易于扩展和维护
2. **模块化设计**：组件之间松耦合，可独立测试和复用
3. **类型安全**：使用 TypeScript 确保类型安全
4. **用户体验**：根据邮箱类型提供最合适的编辑界面
5. **代码复用**：最大化复用现有组件和逻辑
