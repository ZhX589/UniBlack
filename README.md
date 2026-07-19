# UniBlack: 一个可复用的通用云黑系统

[![License](https://shields.io/badge/license-MIT-green)](https://github.com/ZhX589/UniBlack)

UniBlack 是一个可复用的通用云黑系统，支持在线提交&查询、通用查询API、申诉、管理员审核和追责。

> 其中“云黑名单”指的是一个统计曾骗人或做了其他在这个圈子内不被接受的事情的人及其相关社交账号的名单。
> 这仅仅是一个某社区在某时间，对某账号作出了某处理，并公开处理依据、证据、申诉的渠道。一切信息都来源于用户提交，并通过申诉和追责保证可信度、避免诽谤。

--

## 特性

UniBlack 支持
- [ ] 在线提交云黑信息
- [ ] 支持QQ、账号名等多种信息
- [ ] 通用查询API
- [ ] 申诉&审核
- [ ] 完善的用户系统

## 快速上手

可以按照以下方式进行部署

## 文档

文档可以参考 `./docs`

## 开发

### 技术栈 

本项目采用以下技术栈：

**后端**：
- Go&Echo
- GORM
- PostgreSQL

**前端**:
- React + Next.js
- Tailwind CSS
- shadcn/ui

**认证**:
- JWT + Refresh Token
- OAuth2

### 开发步骤

参考 `docs/development.md`。

### Git 提交指南

#### Git Flow

本项目采用简化修改的Git Flow：
- 功能提交都在 `feature` 下创建子分支，测试后再合并进主分支
- bug修正都在 `fix` 下创建子分支，测试之后再合并进主分支
- 文档更新修正都在 `docs` 下创建子分支

遵循以下步骤：

开发完成  
↓  
PR  
↓  
merge main  
↓  
Tag

> [!IMPORTANT] 注意：
> 一般情况下不在 `main` 分支下提交代码

## LICENSE

本项目采用**MIT LICENSE**授权
