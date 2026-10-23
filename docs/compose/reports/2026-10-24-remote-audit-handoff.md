# 远程审计修复交接（2026-10-24）

## 实际发布范围

所有本阶段提交在 `fix/audit-ledgers-and-domain-logic`，不是 main；未强推、未部署。持续提交记录：

| 提交 | 范围 |
| --- | --- |
| 2cbd067 | 前期累计授权、会话、治理、归档与可信UI修复和回归 |
| c8016d4 | 生产默认MinIO凭证拒绝、DSN不再猜测重写、环境测试恢复 |
| b3262fc | 证据NotFound统一、缺少Event读取策略fail-closed |
| b5e391d | multipart payload/文件引用歧义拒绝 |
| 0dc6dfa | service共享64文件/64MiB预算与临界测试 |
| 0d896ad | 72MiB HTTP预算和413 |
| b805e93 | 单JSON值与尾随数据契约 |
| a2fe5fc | 处罚申诉提交service测试与隔离PG事务测试 |
| cbcacec | 动态发布标签DOM回归 |

最后补JSON null/array/scalar拒绝，仅允许非null对象；保留合法尾随空白。该改动的提交号由Git日志查验，不在提交内容中循环引用自身hash。

## 已验收与未验收

前端最新12文件101/101、typecheck/lint/build退出码0；为RTL/jsdom而非实际浏览器E2E。后端每批全量test/vet/build及相关race均通过，最新非null对象补丁的全量test/vet/build及handler/service race亦全部退出码0。

**任务并未全部完成。**本阶段自动轮次结束不等于安全验收通过：

1. 没有可用postgres/psql/docker命令，且未提供隔离DATABASE_URL；migration/SQL事务/并发用例明确SKIP，没有访问生产库替代。PG特性不可用sqlite测试冒充。
2. 裸`/uploads`与S3直链访问仍需独立授权策略及真实入口验收；当前Event JSON可见性控制不能保护泄漏的物理URL。
3. 完整Playwright、键盘/读屏、代理超时与真实multipart网络组合未重跑。
4. refresh family/一次性消费、跨存储删除一致性、历史快照覆盖与handler剩余内部错误脱敏未全部收束。
5. 生产Compose直接拼接原始POSTGRES_PASSWORD的URL编码问题未修，不能声称DSN驱动测试解决该部署问题。

## 保留的原始未提交修改

AGENTS.md、README.md、backend/cmd/server/main.go、backend/cmd/server/routes_test.go、docker-compose.prod.yml 仍为本阶段开始前存在的独立部署/健康检查变更；审查后未与上述修复混提，也未撤销。需要单独确认部署约定后提交。禁止因本阶段结束而清理/覆盖它们。

## 下一步

先准备明确隔离的PostgreSQL并执行现有迁移/申诉/处罚/版本/重置集成测试，再设计证据受控下载与本地/S3非公开存储策略。涉及现有公开URL兼容与生产配置，应明确约定后实施，不能只关闭静态目录而假装S3同步安全。
