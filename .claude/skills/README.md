# New-API Project Skills

本目录包含new-api项目的Claude Code技能（skills），用于标准化开发流程。

## 可用Skills

### `/docker-test` - Docker测试环境部署

本地Docker部署测试环境的完整指南，包括：

- 标准部署流程
- 多数据库兼容性测试（PostgreSQL、MySQL、SQLite）
- 常见问题排查
- 日志查看和调试
- 性能测试
- 环境清理

**使用场景：**
- 需要在本地Docker环境测试代码
- 验证数据库兼容性
- 排查部署相关问题
- 学习标准部署流程

**快速使用：**
```
/docker-test
```

## 如何使用Skills

在Claude Code中，直接输入skill名称即可调用：

```
/docker-test
```

或者在对话中提及相关主题，AI会自动建议使用相应的skill。

## Skill开发规范

### 文件命名

- 使用kebab-case命名：`skill-name.md`
- 文件名应简洁且描述性强

### 文件结构

每个skill文件应包含以下部分：

```markdown
---
skill: skill-name
description: 简短描述（一句话）
tags: [tag1, tag2, tag3]
---

# Skill标题

简介段落

## 主要内容

### 子章节

...

## 快速参考

常用命令或操作的快速索引
```

### 内容要求

1. **清晰性**：步骤明确，易于理解
2. **完整性**：覆盖常见场景和边界情况
3. **实用性**：提供可直接执行的命令和示例
4. **可维护性**：随项目演进及时更新

### 最佳实践

- 使用代码块展示命令和配置
- 提供故障排查指南
- 包含注意事项和警告
- 添加快速参考部分
- 使用清晰的章节结构

## 添加新Skill

1. 在`.claude/skills/`目录创建新的`.md`文件
2. 按照上述规范编写内容
3. 在本README中添加skill说明
4. 如果skill涉及项目规范，在`CLAUDE.md`中添加引用

## 相关文档

- 项目规范：`/CLAUDE.md`
- Docker测试文档：`/docs/DOCKER_TESTING.md`
