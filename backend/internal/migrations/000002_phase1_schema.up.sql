-- Phase 1: Complete database schema
-- Extends initial schema with all core entities

-- =====================================================
-- 1. Extend users table
-- =====================================================
ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(512);
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMP WITH TIME ZONE;

-- =====================================================
-- 2. Extend roles table
-- =====================================================
ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT false;

-- =====================================================
-- 3. Extend permissions table
-- =====================================================
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS resource VARCHAR(50);
ALTER TABLE permissions ADD COLUMN IF NOT EXISTS action VARCHAR(50);

-- =====================================================
-- 4. Organizations (预留)
-- =====================================================
CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add org_id to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS org_id UUID REFERENCES organizations(id) ON DELETE SET NULL;

-- =====================================================
-- 5. Subjects (被举报对象 - 核心)
-- =====================================================
CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name VARCHAR(255) NOT NULL,
    notes TEXT,
    risk_level SMALLINT DEFAULT 0 CHECK (risk_level >= 0 AND risk_level <= 5),
    case_count INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'cleared', 'archived')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =====================================================
-- 6. Identifiers (标识符)
-- =====================================================
CREATE TABLE IF NOT EXISTS identifiers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL CHECK (type IN ('qq', 'discord', 'telegram', 'minecraft_uuid', 'steam', 'email', 'phone', 'ip', 'other')),
    value VARCHAR(255) NOT NULL,
    is_primary BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(type, value)
);

CREATE INDEX idx_identifier_subject ON identifiers(subject_id);
CREATE INDEX idx_identifier_type_value ON identifiers(type, value);

-- =====================================================
-- 7. Cases (案件)
-- =====================================================
CREATE TABLE IF NOT EXISTS cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'approved', 'rejected', 'closed')),
    severity SMALLINT DEFAULT 1 CHECK (severity >= 1 AND severity <= 5),
    verdict TEXT,
    submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    closed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_case_subject ON cases(subject_id);
CREATE INDEX idx_case_status ON cases(status);
CREATE INDEX idx_case_submitted_by ON cases(submitted_by);

-- =====================================================
-- 8. Evidence (证据)
-- =====================================================
CREATE TABLE IF NOT EXISTS evidence (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('image', 'file', 'link', 'text')),
    title VARCHAR(255),
    description TEXT,
    url VARCHAR(512),
    file_size BIGINT,
    sha256 VARCHAR(64),
    mime_type VARCHAR(100),
    uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_evidence_case ON evidence(case_id);

-- =====================================================
-- 9. Submissions (举报提交)
-- =====================================================
CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID REFERENCES cases(id) ON DELETE SET NULL,
    subject_identifiers JSONB NOT NULL DEFAULT '[]',
    reason TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'pending', 'approved', 'rejected')),
    submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    review_notes TEXT,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_submission_status ON submissions(status);
CREATE INDEX idx_submission_submitted_by ON submissions(submitted_by);

-- =====================================================
-- 10. Appeals (申诉)
-- =====================================================
CREATE TABLE IF NOT EXISTS appeals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id UUID NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    submitted_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    review_notes TEXT,
    reviewed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_appeal_case ON appeals(case_id);
CREATE INDEX idx_appeal_status ON appeals(status);

-- =====================================================
-- 11. AuditLog (审计日志)
-- =====================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID,
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_auditlog_user ON audit_logs(user_id);
CREATE INDEX idx_auditlog_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_auditlog_created ON audit_logs(created_at);

-- =====================================================
-- 12. Default data
-- =====================================================

-- Default roles
INSERT INTO roles (name, description, is_system) VALUES
    ('admin', '系统管理员，拥有所有权限', true),
    ('moderator', '审核员，可以审核案件和申诉', true),
    ('user', '普通用户，可以提交举报', true)
ON CONFLICT (name) DO NOTHING;

-- Default permissions
INSERT INTO permissions (name, resource, action, description) VALUES
    ('subject:create', 'subject', 'create', '创建被举报对象'),
    ('subject:read', 'subject', 'read', '查看被举报对象'),
    ('subject:update', 'subject', 'update', '更新被举报对象'),
    ('subject:delete', 'subject', 'delete', '删除被举报对象'),
    ('case:create', 'case', 'create', '创建案件'),
    ('case:read', 'case', 'read', '查看案件'),
    ('case:update', 'case', 'update', '更新案件'),
    ('case:delete', 'case', 'delete', '删除案件'),
    ('case:review', 'case', 'review', '审核案件'),
    ('evidence:create', 'evidence', 'create', '上传证据'),
    ('evidence:read', 'evidence', 'read', '查看证据'),
    ('submission:create', 'submission', 'create', '提交举报'),
    ('submission:read', 'submission', 'read', '查看举报'),
    ('submission:review', 'submission', 'review', '审核举报'),
    ('appeal:create', 'appeal', 'create', '提交申诉'),
    ('appeal:read', 'appeal', 'read', '查看申诉'),
    ('appeal:review', 'appeal', 'review', '审核申诉'),
    ('user:read', 'user', 'read', '查看用户'),
    ('user:update', 'user', 'update', '更新用户'),
    ('audit:read', 'audit', 'read', '查看审计日志')
ON CONFLICT (name) DO NOTHING;

-- Assign all permissions to admin role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- Assign moderation permissions to moderator role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'moderator'
AND p.name IN (
    'subject:read', 'subject:update',
    'case:read', 'case:update', 'case:review',
    'evidence:read',
    'submission:read', 'submission:review',
    'appeal:read', 'appeal:review',
    'user:read'
)
ON CONFLICT DO NOTHING;

-- Assign basic permissions to user role
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'user'
AND p.name IN (
    'subject:read',
    'case:read',
    'evidence:read',
    'submission:create', 'submission:read',
    'appeal:create', 'appeal:read'
)
ON CONFLICT DO NOTHING;
