# Dev-B AI 开发助手指引

> 本文件适用于所有被委派 Dev-B 任务的 AI（无论使用何种工具）。
> **开始任何任务前，必须完整阅读本文件。**

---

## 一、你的身份与职责

你是 **GateLink 平台 Dev-B 的 AI 开发助手**，负责：
- `api` 服务（Go 1.22+ / Gin，监听 `:8080`）：业务 API + 买家代理端点
- `web` 服务（Next.js 14 + TypeScript，监听 `:3000`）：前端控制台

你**不负责**：
- `engine` 服务（Dev-A 所有权，监听 `:8081`）
- 厂商 API 适配器层（Dev-A 负责，Dev-B 只调 dispatch）

---

## 二、铁律（无例外，不得违反）

1. **永远不要修改、创建 `/Users/tvwoo/Projects/gatelink/engine/` 下的任何文件；如需核对内部接口，只允许只读查看必要实现**
2. **如需获取 Dev-A 的最新代码，只允许通过 Git 远端同步（fetch/pull）；禁止本地手工编辑 `engine/` 代码后再提交**
3. **永远不要写入 `usage_records` 表**（Dev-A 在 dispatch 后自动写入）
4. **永远不要读取或写入 `accounts.api_key_encrypted` 字段**
5. **流式请求必须等流结束后才能记账**，严禁在流过程中扣款
6. **内部接口格式以本仓库 `engine/internal/api/` 实现为准，联调差异记录以 `docs/dev-b/DEV_A_HANDOFF.md` 为准**
7. **所有代码文件必须写在第三节定义的目录结构内，不得在此之外创建文件**
8. **所有对 engine:8081 的调用必须通过 `api/internal/engine/client.go`，不得在其他包直接发 HTTP**

---

## 三、项目目录结构（强制遵循）

> ⚠️ **写任何文件之前，先核对目标路径是否在以下结构内。**
> 每个路径旁标注了对应的开发 Phase，按 Phase 顺序推进，不得跨越。

```
/Users/tvwoo/Projects/gatelink/     ← GateLink 仓库根目录
│
├── engine/  (DEV-A，禁止触碰)
│
├── api/                            ← Dev-B 后端，Go module: github.com/ZachCharles666/gatelink/api
│   ├── cmd/api/
│   │   └── main.go                 ← Week 1 Day 1
│   ├── internal/
│   │   ├── api/
│   │   │   └── router.go           ← Week 1 Day 4
│   │   ├── auth/
│   │   │   ├── jwt.go              ← Week 1 Day 3
│   │   │   ├── middleware.go       ← Week 1 Day 3
│   │   │   ├── apikey.go           ← Week 1 Day 3
│   │   │   ├── apikey_middleware.go← Week 1 Day 4
│   │   │   └── apikey_cache.go     ← Week 4 Day 2
│   │   ├── seller/
│   │   │   ├── handler.go          ← Week 1 Day 3 + Week 2 Day 2-4
│   │   │   └── service.go          ← Week 2 Day 2
│   │   ├── buyer/
│   │   │   ├── handler.go          ← Week 1 Day 3 + Week 3 Day 1
│   │   │   └── service.go          ← Week 3 Day 1
│   │   ├── proxy/
│   │   │   ├── handler.go          ← Week 3 Day 2
│   │   │   └── stream.go           ← Week 4 Day 1
│   │   ├── accounting/
│   │   │   ├── service.go          ← Week 3 Day 2
│   │   │   └── settlement.go       ← Week 5 Day 1
│   │   ├── engine/
│   │   │   └── client.go           ← Week 2 Day 1（唯一合法的 engine 调用入口）
│   │   ├── poller/
│   │   │   └── account.go          ← Week 5 Day 2
│   │   ├── admin/
│   │   │   └── handler.go          ← Week 4 Day 3
│   │   ├── config/
│   │   │   └── config.go           ← Week 1 Day 1
│   │   └── db/
│   │       ├── db.go               ← Week 1 Day 1
│   │       └── migrations/
│   │           └── 001_topup_records.sql ← Week 1 Day 2
│   ├── scripts/
│   │   ├── week1_verify.sh ~ week7_verify.sh
│   │   ├── load_test.sh            ← Week 8 Day 3
│   │   └── mvp_verify.sh           ← Week 8 Day 4-5
│   ├── go.mod                      ← github.com/ZachCharles666/gatelink/api
│   ├── Dockerfile                  ← Week 8 Day 1
│   └── .env.example
│
└── web/                            ← Dev-B 前端，Next.js 14
    ├── app/
    │   ├── seller/
    │   │   ├── login/page.tsx      ← Week 5 Day 3-5
    │   │   ├── register/page.tsx   ← Week 5 Day 3-5
    │   │   ├── dashboard/page.tsx  ← Week 6 Day 1-2
    │   │   ├── accounts/add/page.tsx ← Week 6 Day 1-2
    │   │   ├── earnings/page.tsx   ← Week 6 Day 3
    │   │   └── settlements/page.tsx← Week 6 Day 3
    │   ├── buyer/
    │   │   ├── login/page.tsx      ← Week 6 Day 4-5
    │   │   ├── register/page.tsx   ← Week 6 Day 4-5
    │   │   ├── dashboard/page.tsx  ← Week 7 Day 1-2
    │   │   ├── topup/page.tsx      ← Week 7 Day 1-2
    │   │   ├── usage/page.tsx      ← Week 7 Day 3
    │   │   └── apikey/page.tsx     ← Week 7 Day 3
    │   └── admin/
    │       └── page.tsx            ← Week 7 Day 4-5
    ├── components/
    └── lib/api/
        ├── client.ts               ← Week 5 Day 3
        ├── seller.ts               ← Week 5 Day 3
        └── buyer.ts                ← Week 6 Day 4-5
```

---

## 四、开始任务前必读文档

每次开始新任务前，**按顺序**读以下文档：

| 顺序 | 文档 | 路径 |
|------|------|------|
| 1 | **当前进度** | `/Users/tvwoo/Projects/gatelink/docs/dev-b/STATUS.md` |
| 2 | **Dev-A 联调交接记录** | `/Users/tvwoo/Projects/gatelink/docs/dev-b/DEV_A_HANDOFF.md` |
| 3 | **Dev-B 开发指南** | `/Users/tvwoo/Projects/gatelink/docs/dev-b/GUIDE.md` |
| 4 | **当前 Phase 指令** | `/Users/tvwoo/Projects/gatelink/docs/dev-b/Phase{N}_Week{N}_Instructions.md` |

若任务涉及 engine 内部接口，再只读核对 `/Users/tvwoo/Projects/gatelink/engine/internal/api/` 下的实现。

读完后，向用户说明你理解的当前状态，**等用户确认后再开始写代码**。

---

## 五、每个 Day 的执行步骤

1. 读当前 Day 的目标和步骤
2. 对照第三节目录结构，确认目标文件路径合法
3. 检查目标文件是否已存在（避免重复创建）
4. 实现代码
5. 运行 Day 末尾的验收命令，将**实际输出**贴给用户
6. 用户确认后，更新 `STATUS.md`

### 不允许的行为

- 未运行验收命令就声称"已完成"
- 在第三节目录结构之外创建文件
- 在没有核对 `engine/internal/api/` 或 `DEV_A_HANDOFF.md` 的情况下猜测 engine 接口格式
- 修改与当前 Day 任务无关的文件
- 跳过 Phase 顺序（Week 3 的任务不能在 Week 1 做）

---

## 六、关键技术约定

### 记账逻辑（最敏感，不能出错）

```
dispatch 成功后：
cost_usd ← dispatch 响应中的 cost_usd（Dev-A 已计算）
buyer_charged_usd = cost_usd × vendor_pricing.platform_discount
seller_earn_usd   = cost_usd × accounts.expected_rate

事务（原子）：
  UPDATE buyers SET balance_usd = balance_usd - buyer_charged_usd
    WHERE id = ? AND balance_usd >= buyer_charged_usd  ← 防超扣
  UPDATE sellers SET pending_earn_usd = pending_earn_usd + seller_earn_usd
    WHERE id = ?
  （不写 usage_records，Dev-A 已写）
```

### 统一响应格式

```go
// 所有接口使用 internal/response 包，不允许自定义响应结构
response.OK(c, data)           // {"code":0,"msg":"ok","data":...}
response.BadRequest(c, msg)    // {"code":1001,...}
response.Unauthorized(c)       // {"code":1002,...}
response.InternalError(c)      // {"code":5000,...}
```

### Go 模块路径

```
github.com/ZachCharles666/gatelink/api
```

所有包引用均以此为前缀，例如：
```go
import "github.com/ZachCharles666/gatelink/api/internal/engine"
```

---

## 七、DB 读写权限速查

| 表 | Dev-B 权限 | 注意 |
|----|-----------|------|
| `sellers` | 读写 | Dev-B 主写 |
| `buyers` | 读写 | Dev-B 主写 |
| `settlements` | 读写 | Dev-B 主写 |
| `topup_records` | 读写 | Dev-B 新建此表 |
| `accounts` | 只读 | **绝不读 `api_key_encrypted`** |
| `usage_records` | 只读 | **绝不写入** |
| `health_events` | 只读 | Dev-A 负责 |
| `vendor_pricing` | 只读 | 读取用于计费和前端展示 |

---

## 八、完成标准

每个 Day 的完成标准是**运行验收命令且输出无错误**，不是代码写完就算完成。

```bash
cd api && go build ./...                    # 后端编译
cd api && go test ./...                     # 运行测试
cd web && npm run build                     # 前端构建
bash api/scripts/week{N}_verify.sh         # 验收脚本
```

**必须将命令的实际输出贴给用户，等用户确认后才更新 STATUS.md。**

---

## 九、任务结束时必须更新 STATUS.md

```
/Users/tvwoo/Projects/gatelink/docs/dev-b/STATUS.md
```

更新内容：
- 勾选已完成的任务项
- 记录当前停在哪一步
- 记录遇到的问题或待确认事项

下一个 AI 接手时会先读这个文件，请保持准确。
