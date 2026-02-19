#!/usr/bin/env bash
set -euo pipefail

# 说明：在项目根目录执行此脚本，可一键初始化 FlexiRAG Engine 的目录结构。
# 用法：bash scripts/wsl/init_project_structure.sh

mkdir -p \
  cmd/server \
  cmd/worker \
  configs \
  internal/api/v1 \
  internal/api/middleware \
  internal/core/agent_mgmt \
  internal/core/knowledge \
  internal/core/user \
  internal/engine/memory \
  internal/engine/planner \
  internal/tools/crawler \
  internal/tools/calculator \
  internal/data/db \
  internal/data/cache \
  internal/model \
  internal/pkg/llm \
  internal/pkg/vector

# 保留目录结构（空目录占位）
find cmd configs internal -type d -exec touch '{}/.keep' ';'

echo "目录结构初始化完成。"
