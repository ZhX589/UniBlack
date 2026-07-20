# UniBlack Database Design

## 实体关系概览

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│    User      │────▶│  UserRoles  │◀────│    Role     │
└─────────────┘     └─────────────┘     └──────┬──────┘
                                               │
                                               ▼
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│  Permission │◀────│RolePermission│    │ Organization│
└─────────────┘     └─────────────┘     └─────────────┘

┌─────────────┐     ┌─────────────┐
│   Subject   │────▶│ Identifier  │
└──────┬──────┘     └─────────────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│    Case     │────▶│  Evidence   │
└──────┬──────┘     └─────────────┘
       │
       ├───────────────┐
       ▼               ▼
┌─────────────┐  ┌─────────────┐
│  Submission │  │   Appeal    │
└─────────────┘  └─────────────┘

┌─────────────┐
│  AuditLog   │  (记录所有操作)
└─────────────┘
```

## 核心实体

### 1. User (用户)

系统用户，包括普通用户和管理员。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| username | VARCHAR(255) | 用户名，唯一 |
| email | VARCHAR(255) | 邮箱，唯一 |
| password_hash | VARCHAR(255) | 密码哈希 |
| auth_provider | VARCHAR(50) | 认证提供者（local/oauth:github等） |
| external_id | VARCHAR(255) | 外部系统ID |
| display_name | VARCHAR(255) | 显示名称 |
| avatar_url | VARCHAR(512) | 头像URL |
| is_active | BOOLEAN | 是否激活 |
| last_login_at | TIMESTAMP | 最后登录时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 2. Role (角色)

RBAC 角色。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | VARCHAR(50) | 角色名，唯一 |
| description | TEXT | 描述 |
| is_system | BOOLEAN | 是否系统角色（不可删除） |
| created_at | TIMESTAMP | 创建时间 |

### 3. Permission (权限)

细粒度权限控制。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | VARCHAR(100) | 权限名，唯一 |
| resource | VARCHAR(50) | 资源类型（subject/case/evidence等） |
| action | VARCHAR(50) | 操作（create/read/update/delete） |
| description | TEXT | 描述 |

### 4. UserRoles (用户角色关联)

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | UUID | 外键 → User |
| role_id | UUID | 外键 → Role |

### 5. RolePermissions (角色权限关联)

| 字段 | 类型 | 说明 |
|------|------|------|
| role_id | UUID | 外键 → Role |
| permission_id | UUID | 外键 → Permission |

### 6. Organization (组织) - 预留

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| name | VARCHAR(255) | 组织名 |
| slug | VARCHAR(100) | URL 友好标识 |
| description | TEXT | 描述 |
| created_at | TIMESTAMP | 创建时间 |

### 7. Subject (被举报对象) - 核心

被举报的人或账号，是系统的核心实体。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| display_name | VARCHAR(255) | 显示名称 |
| notes | TEXT | 备注 |
| risk_level | SMALLINT | 风险等级（0-5） |
| case_count | INTEGER | 关联案件数 |
| status | VARCHAR(20) | 状态（active/cleared/archived） |
| created_by | UUID | 创建者 → User |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 8. Identifier (标识符)

Subject 的各种标识符，支持多种类型。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| subject_id | UUID | 外键 → Subject |
| type | VARCHAR(50) | 类型（qq/discord/telegram/minecraft_uuid/steam/email/phone/ip） |
| value | VARCHAR(255) | 标识符值 |
| is_primary | BOOLEAN | 是否主要标识符 |
| created_at | TIMESTAMP | 创建时间 |

**约束**: (type, value) 唯一

### 9. Case (案件)

针对 Subject 的处理案件。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| subject_id | UUID | 外键 → Subject |
| title | VARCHAR(255) | 案件标题 |
| description | TEXT | 案件描述 |
| status | VARCHAR(20) | 状态（draft/pending/approved/rejected/closed） |
| severity | SMALLINT | 严重程度（1-5） |
| verdict | TEXT | 裁定结果 |
| submitted_by | UUID | 提交者 → User |
| reviewed_by | UUID | 审核者 → User |
| reviewed_at | TIMESTAMP | 审核时间 |
| closed_at | TIMESTAMP | 关闭时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 10. Evidence (证据)

案件的证据材料。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| case_id | UUID | 外键 → Case |
| type | VARCHAR(20) | 类型（image/file/link/text） |
| title | VARCHAR(255) | 标题 |
| description | TEXT | 描述 |
| url | VARCHAR(512) | 文件URL或链接 |
| file_size | BIGINT | 文件大小（字节） |
| sha256 | VARCHAR(64) | 文件SHA256哈希 |
| mime_type | VARCHAR(100) | MIME类型 |
| uploaded_by | UUID | 上传者 → User |
| created_at | TIMESTAMP | 创建时间 |

### 11. Submission (举报提交)

用户提交的举报请求。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| case_id | UUID | 外键 → Case（审核通过后关联） |
| subject_identifiers | JSONB | 提交的标识符列表 |
| reason | TEXT | 举报原因 |
| status | VARCHAR(20) | 状态（draft/pending/approved/rejected） |
| submitted_by | UUID | 提交者 → User |
| reviewed_by | UUID | 审核者 → User |
| review_notes | TEXT | 审核备注 |
| reviewed_at | TIMESTAMP | 审核时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 12. Appeal (申诉)

对案件的申诉。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| case_id | UUID | 外键 → Case |
| reason | TEXT | 申诉理由 |
| status | VARCHAR(20) | 状态（pending/approved/rejected） |
| submitted_by | UUID | 申诉者 → User |
| reviewed_by | UUID | 审核者 → User |
| review_notes | TEXT | 审核备注 |
| reviewed_at | TIMESTAMP | 审核时间 |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 13. AuditLog (审计日志)

记录所有重要操作。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| user_id | UUID | 操作者 → User |
| action | VARCHAR(50) | 操作类型（create/update/delete/login等） |
| resource_type | VARCHAR(50) | 资源类型 |
| resource_id | UUID | 资源ID |
| changes | JSONB | 变更内容 |
| ip_address | INET | IP地址 |
| user_agent | TEXT | User-Agent |
| created_at | TIMESTAMP | 创建时间 |

## 索引策略

```sql
-- Identifier 查询优化
CREATE UNIQUE INDEX idx_identifier_type_value ON identifiers(type, value);
CREATE INDEX idx_identifier_subject ON identifiers(subject_id);

-- Case 查询优化
CREATE INDEX idx_case_subject ON cases(subject_id);
CREATE INDEX idx_case_status ON cases(status);
CREATE INDEX idx_case_submitted_by ON cases(submitted_by);

-- Evidence 查询优化
CREATE INDEX idx_evidence_case ON evidence(case_id);

-- Submission 查询优化
CREATE INDEX idx_submission_status ON submissions(status);
CREATE INDEX idx_submission_submitted_by ON submissions(submitted_by);

-- Appeal 查询优化
CREATE INDEX idx_appeal_case ON appeals(case_id);
CREATE INDEX idx_appeal_status ON appeals(status);

-- AuditLog 查询优化
CREATE INDEX idx_auditlog_user ON audit_logs(user_id);
CREATE INDEX idx_auditlog_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_auditlog_created ON audit_logs(created_at);
```