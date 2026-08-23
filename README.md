# Food Allergen Cross-contact Analyzer

食品工厂质量与配方团队使用的内部交叉接触风险分析台。系统维护过敏原谱、工艺路线和有向接触关系，以可解释的带权路径传播生成矩阵，并保存不可覆盖的评估与复核记录。

> 本系统仅用于内部风险分析，不替代法规判断、实验室检测、最终标签批准，也不控制生产设备。

## Docker 快速启动

```bash
cp .env.example .env
docker compose up -d --build
docker compose ps
```

服务健康后访问：

- 前端：<http://127.0.0.1:18525>
- 后端健康：<http://127.0.0.1:19525/healthz>
- 后端就绪：<http://127.0.0.1:19525/readyz>
- PostgreSQL：`127.0.0.1:57525`

默认测试账号：

| 角色 | 用户名 | 密码 | 权限 |
| --- | --- | --- | --- |
| `quality_analyst` | `analyst` | `Analyst#525` | 维护输入、提交并运行评估 |
| `reviewer` | `reviewer` | `Reviewer#525` | 只读输入、接受或拒绝待复核评估、检索审计 |
| `admin` | `admin` | `Admin#525Secure` | 输入维护与复核管理 |

停止并删除本项目数据卷：

```bash
docker compose down -v --remove-orphans
```

## 主要功能

- 过敏原谱：维护材料名称、过敏原集合、证据来源、供应商声明日期、状态与版本；详情显示路线引用。
- 工艺路线：维护有序步骤、每步引用的过敏原谱、声明过敏原、路线状态和乐观锁版本。
- 接触关系：在路线内维护来源/目标步骤、接触类型、共享设备、清洗衰减、带入概率和证据记录。
- 交叉接触矩阵：从真实路线、谱和已启用接触边计算目标步骤 × 过敏原矩阵，可按过敏原筛选并检查完整证据路径。
- 评估工作台：执行 `queued -> calculating -> pending_review -> accepted | rejected` 状态流；输入变化使旧结果变为 `stale`。
- 审计检索：记录 request ID、操作者、实体、动作、前后摘要和版本元数据，并提供版本摘要对比。

系统不包含订单、库存、采购、财务、电商、通用质量工单或生产设备控制。

## 技术栈

- 后端：Go 1.22、Gin、GORM、validator/v10、JWT、bcrypt
- 正式数据库：PostgreSQL 16
- 运行冒烟与普通测试：GORM SQLite（仅限 `runtime_smoke.json` 和测试）
- 前端：Vue 3、TypeScript、Vite、Element Plus、Pinia、Vue Router、Lucide
- 部署：Docker Compose、Nginx SPA 与 `/api` 反向代理

## 目录

```text
.
├── backend/
│   ├── cmd/server/                 # 启动、迁移、注入与优雅停机
│   └── internal/
│       ├── analyzer/               # 图构建、传播、评分、阈值
│       ├── config/                 # 环境配置与约束
│       ├── constants/              # 风险与评估共享枚举
│       ├── dto/                    # 请求、查询和页面 DTO
│       ├── handler/                # HTTP 绑定与统一响应
│       ├── middleware/             # 请求 ID、日志、认证、RBAC、恢复、限流
│       ├── model/                  # GORM 实体与数据库约束
│       ├── repository/             # 查询、条件更新和事务
│       ├── router/                 # `/api/v1` 路由
│       ├── service/                # 业务校验、版本与评估编排
│       └── util/                   # 响应、分页和校验错误
├── frontend/src/
│   ├── api/                        # Axios 客户端和领域 API
│   ├── components/common/          # 风险、证据、版本共享组件
│   ├── hooks/                      # 认证与评估轮询
│   ├── pages/                      # 五个业务页面及登录页
│   ├── router/                     # 路由和认证守卫
│   ├── stores/                     # 认证与评估 Pinia 状态
│   ├── types/                      # 共享枚举和领域类型
│   └── utils/                      # 格式化函数
├── docker-compose.yml
├── go.work
└── runtime_smoke.json              # 仅包含 SQLite 启动 manifest
```

## 核心实体

### `AllergenProfile`

分析输入而非库存物料。`allergens_json` 始终保存规范化后的 JSON 数组；更新使用 `expected_version` 条件写入并令既有评估失效。

### `ProcessRoute`

`ordered_steps_json` 的每个步骤包含 `step_code`、`step_name`、`profile_id`。步骤代码在路线内唯一，active 路线只能引用 active 谱。

### `ContactEdge`

边属于单一路线。端点必须来自该路线，禁止自环；有向环路可以由多条边形成，但传播器按路径访问状态安全终止。

### `AssessmentRun`

每次运行保存完整输入快照、矩阵、风险项、算法/阈值版本和完成时间。没有结果更新接口；复核仅条件更新状态与复核字段。新输入版本不会覆盖旧 JSON，而是把旧结果标记为 `stale`。

## 传播算法

1. 从 `ProcessRoute.ordered_steps_json` 建立节点，从同路线且 `enabled=true` 的 `ContactEdge` 建立有向边。
2. 每个步骤引用的 `AllergenProfile` 为过敏原传播种子。
3. 每条边计算：`edge_weight = (1 - cleaning_factor) * carryover_probability`。
4. 路径累计：`path_score = previous_score * edge_weight`，保留六位小数，不使用材料名字符串匹配替代图算法。
5. 每条结果保留来源谱/材料、来源和目标步骤、完整路径、累计原始分数、关键边、逐边清洗证据和阈值版本。
6. 每条路径持有独立访问集合；回访节点时跳过该边。达到 `MAX_PROPAGATION_DEPTH` 后停止继续展开并记录次数。
7. 声明过敏原和传播过敏原分别输出；矩阵中的 `declared` 只表示路线声明状态，不生成法规标签结论。

默认风险阈值：

| 等级 | 条件 |
| --- | --- |
| `low` | 分数 `< 0.12` |
| `medium` | 分数 `>= 0.12` 且 `< 0.35` |
| `high` | 分数 `>= 0.35` 且 `< 0.65` |
| `critical` | 分数 `>= 0.65` |

阈值和 `RISK_THRESHOLD_VERSION` 会进入每次评估输入快照。

## 状态机与并发

```text
queued -> calculating -> pending_review -> accepted
                              \-------> rejected
pending_review | accepted | rejected --输入变化--> stale
```

- 运行使用 `WHERE id = ? AND assessment_status = 'queued'` 条件更新。
- 完成使用 `calculating` 条件更新并在同一事务内写审计。
- 接受/拒绝使用 `pending_review` 条件更新，拒绝和接受均要求复核理由。
- 谱、路线和边使用 `expected_version` 乐观锁；版本不一致返回 HTTP `409 version_conflict`。
- 输入版本更新、边新增或更新在事务内将关联旧结果改为 `stale` 并写审计。

## 共享枚举位置

### `RiskLevel = low | medium | high | critical`

| 层 | 位置 |
| --- | --- |
| 后端常量 | `backend/internal/constants/risk.go` |
| 数据库约束 / 模型 | `backend/internal/model/assessment_run.go` 的 `highest_risk_level` CHECK |
| 图结果与阈值映射 | `backend/internal/analyzer/propagate.go`、`threshold.go` |
| 仓储持久化 | `backend/internal/repository/assessment_repository.go` |
| 服务编排 | `backend/internal/service/assessment_service.go` |
| 前端类型 | `frontend/src/types/risk.ts`、`types/assessment.ts`、`types/domain.ts` |
| 前端 store | `frontend/src/stores/assessments.ts` 的风险计数 |
| 共享组件 | `frontend/src/components/common/RiskBadge.vue`、`EvidencePathPanel.vue` |
| 页面 | `frontend/src/pages/MatrixPage.vue`、`AssessmentsPage.vue` |

### `AssessmentStatus = queued | calculating | pending_review | accepted | rejected | stale`

| 层 | 位置 |
| --- | --- |
| 后端常量 / 状态转换 | `backend/internal/constants/assessment.go` |
| 数据库约束 / 模型 | `backend/internal/model/assessment_run.go` 的 `assessment_status` CHECK |
| DTO 查询 | `backend/internal/dto/assessment.go` |
| 仓储条件更新 | `backend/internal/repository/assessment_repository.go`、`support_repository.go` |
| 服务编排 | `backend/internal/service/assessment_service.go` |
| 路由权限 | `backend/internal/router/assessment_router.go` |
| 前端类型 | `frontend/src/types/assessment.ts` |
| 前端 store | `frontend/src/stores/assessments.ts` |
| 页面 | `frontend/src/pages/AssessmentsPage.vue` |

## 权限

| 操作 | quality_analyst | reviewer | admin |
| --- | :---: | :---: | :---: |
| 查看谱、路线、矩阵、评估 | ✓ | ✓ | ✓ |
| 新建/更新谱、路线、边 | ✓ |  | ✓ |
| 提交/运行评估 | ✓ |  | ✓ |
| 接受/拒绝评估 |  | ✓ | ✓ |
| 审计检索 |  | ✓ | ✓ |

后端 JWT/RBAC 是权限边界；前端守卫和按钮显隐仅改善交互，不替代后端校验。服务不信任客户端角色头。

## API

所有业务 API 前缀为 `/api/v1`，除登录外均要求 `Authorization: Bearer <token>`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/auth/login` | 登录 |
| `GET` | `/auth/me` | 当前用户 |
| `GET/POST` | `/profiles` | 查询 / 创建谱 |
| `GET/PUT` | `/profiles/:id` | 详情与路线引用 / 新版本 |
| `GET/POST` | `/routes` | 查询 / 创建路线 |
| `GET/PUT` | `/routes/:id` | 详情 / 新版本 |
| `GET/POST` | `/contact-edges` | 查询 / 创建接触边 |
| `GET/PUT` | `/contact-edges/:id` | 详情 / 新版本 |
| `POST` | `/matrix/compute` | 计算但不持久化矩阵 |
| `GET/POST` | `/assessments` | 查询 / 入队 |
| `GET` | `/assessments/:id` | 不可变结果详情 |
| `POST` | `/assessments/:id/run` | 条件运行 |
| `POST` | `/assessments/:id/review` | reviewer 接受或拒绝 |
| `GET` | `/audit` | 审计检索 |
| `GET` | `/versions/:entityType/:id?version=n` | 最近版本变更摘要 |

统一成功响应含 `success`、`data`、`request_id`，分页响应另含 `meta`。统一错误含 `error.code`、`error.message`、可选 `error.details` 和 `request_id`。常见状态码为 `401`、`403`、`404`、`409`、`422`、`429`。

## 环境变量

| 变量 | 默认值 / 用途 |
| --- | --- |
| `COMPOSE_PROJECT_NAME` | `food-allergen-crosscontact-analyzer` |
| `FRONTEND_PORT` | `18525` |
| `BACKEND_PORT` | `19525` |
| `DB_PORT` | `57525` |
| `POSTGRES_DB/USER/PASSWORD` | Compose PostgreSQL 凭据 |
| `DB_DRIVER` | Compose 固定 `postgres`；runtime smoke 为 `sqlite` |
| `DB_DSN` | GORM 连接串 |
| `DB_AUTO_MIGRATE` | 启动时迁移；默认 `true` |
| `JWT_SECRET` | 至少 32 字节，生产环境必须替换 |
| `JWT_EXPIRY_HOURS` | JWT 有效期，默认 12 小时 |
| `CORS_ORIGINS` | 逗号分隔的允许来源 |
| `MAX_PROPAGATION_DEPTH` | 传播最大边数，范围 2–64 |
| `RISK_THRESHOLD_MEDIUM/HIGH/CRITICAL` | 严格递增的风险阈值 |
| `RISK_THRESHOLD_VERSION` | 进入输入快照的阈值版本 |
| `RATE_LIMIT_PER_MINUTE` | 每客户端 IP 的本地分钟限流 |
| `LOG_LEVEL` | `info` 或 `debug` |

## 本地开发与验证

后端（SQLite 示例）：

```bash
cd backend
PORT=20525 \
DB_DRIVER=sqlite \
DB_DSN='file:local-dev?mode=memory&cache=shared' \
DB_AUTO_MIGRATE=true \
JWT_SECRET='local-development-secret-at-least-32-bytes' \
go run ./cmd/server
```

前端：

```bash
npm --prefix frontend ci
npm --prefix frontend run dev
```

完整静态与测试验证：

```bash
go work sync
go build ./backend/...
go vet ./backend/...
go test ./backend/...
npm --prefix frontend ci
npm --prefix frontend run build
python3 /Users/gaobo/.codex/skills/go-annotation-pipeline/scripts/project_scale.py .
python3 /Users/gaobo/.codex/skills/go-annotation-pipeline/scripts/runtime_smoke.py .
docker compose config --quiet
```

表驱动测试覆盖边权衰减、两段传播、环路跳过、深度上限、阈值边界和状态机合法/非法转换。

## 安全与审计

- JWT 使用 HS256、固定 issuer/audience、到期校验；密码以 bcrypt 保存。
- `RequestID`、`AccessLog`、`Auth`、`RBAC`、`Recovery`、`RateLimit` 为独立中间件函数。
- 访问日志只记录方法、路由模板、状态、耗时、客户端 IP、操作者和 request ID，不记录请求体或查询内容，因此供应商证据备注不会进入访问日志。
- 审计摘要会保存业务字段和证据内容；数据库与审计访问应按组织敏感信息制度管控。
- Nginx 设置 `nosniff`、同源 referrer 与拒绝 framing；生产环境仍应在可信反向代理上启用 TLS。

## 排错

- `readyz` 返回 503：执行 `docker compose logs postgres backend`，确认数据库凭据和健康检查。
- 前端登录后返回 401：清除浏览器中 `allergen_token` 后重新登录，并检查 `JWT_SECRET` 未在运行中变化。
- API 返回 `version_conflict`：刷新页面，基于最新 `version` 重新提交。
- API 返回 `state_conflict`：评估已被运行、复核或因输入变化过期，读取最新评估状态。
- 矩阵为空：确认路线至少有两个有效步骤，并存在 `enabled=true` 且边权大于 0 的接触边。
- Compose 端口冲突：只在确认任务端口分配后调整 `.env`；默认端口为 `18525/19525/57525`。

## License

MIT，见 [LICENSE](./LICENSE)。
