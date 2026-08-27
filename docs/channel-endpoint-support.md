# 渠道端点支持矩阵

本文档记录各渠道类型在 New API 默认请求流程中的端点支持状态，也是新增或修改渠道类型时必须同步维护的能力清单。

## 适用范围

- 以仓库当前代码为准，不以供应商宣传页或路由是否存在作为唯一判断依据。
- “支持”要求默认流程完整具备路由、请求转换、上游 URL、请求发送和响应处理；仅注册 Gin 路由不算支持。
- 本文档按全局和渠道级 `PassThroughBodyEnabled` 均关闭的默认行为记录。开启透传后，上游可能接受更多端点，但不改变本表的默认支持结论。
- 异步图像默认关闭；表中的“异步图像（配置）”仅在渠道高级设置显式开启并选择与上游匹配的 Ali、New API 或 Grsai 协议后可用。
- 模型、API 版本、区域、密钥类型或自定义 Base URL 造成的供应商限制，记录为“条件支持”。
- `GET /v1/models` 等模型目录接口主要返回本系统可用模型，不代表所选上游渠道支持表中的所有推理端点。

## 状态定义

| 状态 | 含义 |
| --- | --- |
| 支持 | 默认流程的请求转换、URL 构造和响应处理完整 |
| 条件 | 仅特定模型、配置、Base URL 或上游兼容实现可用 |
| 透传 | 代码按 OpenAI 兼容路径转发，但项目不保证目标供应商实现该端点 |
| 不支持 | 缺少转换器、URL、响应处理或任务适配器，或代码明确拒绝 |

## 端点缩写

| 缩写 | 客户端入口 |
| --- | --- |
| Chat | `POST /v1/chat/completions` |
| Comp | `POST /v1/completions` |
| Msg | `POST /v1/messages` |
| Resp | `POST /v1/responses` |
| Compact | `POST /v1/responses/compact` |
| Alpha | `POST /v1/alpha/search` |
| Gemini | `POST /v1beta/models/*` 或 `POST /v1/models/*` |
| Emb | `POST /v1/embeddings`，以及兼容的 Gemini embedding 路由 |
| Rank | `POST /v1/rerank` |
| ImgG | `POST /v1/images/generations` |
| ImgE | `POST /v1/images/edits` |
| TTS | `POST /v1/audio/speech` |
| STT | `POST /v1/audio/transcriptions`、`POST /v1/audio/translations` |
| Mod | `POST /v1/moderations` |
| RT | `GET /v1/realtime` WebSocket |

## 当前可选渠道类型

下表必须保持每个前端可选渠道类型一行。端点后缀“条件”或“透传”按上面的状态定义解释；未列出的标准端点均视为不支持。

| ID | 渠道类型 | 文本与协议 | 向量与排序 | 图像与音频 | 实时与任务 | 关键限制 |
| ---: | --- | --- | --- | --- | --- | --- |
| 1 | OpenAI | Chat、Comp、Msg、Resp、Compact、Gemini、Mod | Emb、Rank（透传） | ImgG、ImgE、TTS、STT | RT、OpenAI Video、异步图像（配置） | 官方上游是否实现 Rank 由上游决定；支持完整 OpenAI 视频任务 |
| 2 | MjProxy | - | - | - | Midjourney | 仅使用 `/mj` 专用任务路由 |
| 3 | Azure | Chat、Comp、Msg、Resp、Compact、Gemini、Mod | Emb、Rank（透传） | ImgG、ImgE、TTS、STT（条件） | RT、异步图像（配置） | 使用 Azure deployment 和 API version URL；无视频任务适配器 |
| 4 | Ollama | Chat、Comp、Msg | Emb | - | - | 分别映射到 `/api/chat`、`/api/generate`、`/api/embed` |
| 5 | MjProxyPlus | - | - | - | Midjourney | 仅使用 `/mj` 专用任务路由 |
| 7 | OhMyGPT | OpenAI 兼容端点（透传，含 Compact） | 透传 | 透传 | RT（透传）、异步图像（配置） | 使用 OpenAI 适配器；实际能力由上游决定；无视频任务 |
| 8 | Custom | OpenAI 兼容端点（条件） | 条件 | 条件 | RT（条件）、异步图像（配置） | 所有请求发送到填写的同一个完整 URL，不自动拼接入口路径；无视频任务 |
| 14 | Anthropic | Chat、Msg | - | - | - | Chat 转换为 Anthropic Messages；无 Responses、Embedding 和任务端点 |
| 15 | Baidu | Chat | Emb | - | - | 旧千帆适配器 |
| 16 | Zhipu | Chat | - | - | - | 旧 `/api/paas/v3/model-api` 适配器 |
| 17 | Ali | Chat、Comp、Msg（条件）、Resp | Emb、Rank | ImgG、ImgE | Ali Video、异步图像（配置） | Anthropic Messages 仅配置匹配模型；不支持 Compact 和 Audio |
| 18 | Xunfei | Chat | - | - | - | 使用讯飞专用 WebSocket 请求链 |
| 19 | 360 | OpenAI 兼容端点（透传，含 Compact） | 透传 | 透传 | RT（透传）、异步图像（配置） | 实际能力由上游决定；无视频任务 |
| 20 | OpenRouter | OpenAI 兼容端点（透传，不含 Compact） | 透传 | 透传 | RT（透传）、异步图像（配置） | `APITypeOpenRouter` 不在 Compact 白名单；无视频任务 |
| 22 | FastGPT | OpenAI 兼容端点（透传，含 Compact） | 透传 | 透传 | RT（透传）、异步图像（配置） | 实际能力由上游决定；无视频任务 |
| 23 | Tencent | Chat | - | - | - | 腾讯混元签名请求链 |
| 24 | Gemini | Chat、Msg、Resp、Gemini | Emb | ImgG（仅 Imagen） | Gemini/Veo Video、异步图像（配置） | 无 Compact、ImgE、Audio 和 Rank |
| 25 | Moonshot | Chat、Comp、Msg | Emb、Rank | - | - | Responses 和 Audio 未实现 |
| 26 | Zhipu V4 | Chat、Msg | Emb | ImgG | 异步图像（配置） | 无 Responses、Audio 和 Rank |
| 27 | Perplexity | Chat、Msg、Resp | - | - | - | 无 Compact 及其他媒体端点 |
| 31 | LingYiWanWu | OpenAI 兼容端点（透传，含 Compact） | 透传 | 透传 | RT（透传）、异步图像（配置） | 实际能力由上游决定；无视频任务 |
| 33 | AWS | Chat、Msg | - | - | - | Bedrock Claude/Nova 转换；无 Responses、Embedding 和任务端点 |
| 34 | Cohere | Chat | Rank | - | - | 无 Embedding、Responses 和媒体端点 |
| 35 | MiniMax | Chat、Msg | - | ImgG、TTS | MiniMax/Hailuo Video、异步图像（配置） | Embedding 有请求函数但 URL 构造不支持，因此不计为支持 |
| 36 | SunoAPI | - | - | - | Suno | 仅使用 `/suno` 专用任务路由 |
| 37 | Dify | Chat | - | - | - | 默认映射到 Dify chat-messages |
| 38 | Jina | - | Emb、Rank | - | - | 仅向量和重排 |
| 39 | Cloudflare | Chat、Resp | Emb | STT | - | Audio 仅转录/翻译；无 Img、TTS；Comp/Rank 响应链不完整 |
| 40 | SiliconFlow | Chat、Comp、Msg | Emb、Rank | ImgG、ImgE、TTS、STT | RT（透传）、异步图像（配置） | Responses 未实现；实时能力由上游兼容性决定 |
| 41 | Vertex AI | Chat、Msg、Gemini | - | ImgG（仅 Imagen） | Vertex/Veo Video、异步图像（配置） | 根据模型进入 Claude、Gemini 或 Open-source 模式；无 Responses、Embedding 和 Audio |
| 42 | Mistral | Chat | - | - | - | Embedding、Responses、Messages 和媒体转换未实现 |
| 43 | DeepSeek | Chat、Comp、Msg、Resp | - | - | - | Comp 用于 FIM；无 Compact、Embedding 和媒体端点 |
| 44 | MokaAI | - | Emb | - | - | 其他端点未实现 |
| 45 | VolcEngine | Chat、Msg、Resp | Emb | ImgG、TTS | Doubao Video、异步图像（配置） | ImgE 转换未完成；Rank 请求转换为空，不计为支持 |
| 46 | Baidu V2 | Chat | - | - | - | URL 构造器虽列出 Emb/Img/Rank，但默认转换器未实现 |
| 47 | Xinference | OpenAI 兼容端点（透传，不含 Compact） | 透传 | 透传 | RT（透传）、异步图像（配置） | `APITypeXinference` 不在 Compact 白名单；无视频任务 |
| 48 | xAI | Chat、Resp | - | ImgG、ImgE | 异步图像（配置） | 无 Compact、Messages、Embedding、Audio 和 Video |
| 49 | Coze | Chat | - | - | - | 使用 Coze chat 创建和轮询流程 |
| 50 | Kling | - | - | - | Kling Video | 支持通用视频路由及 `/kling/v1` 专用路由 |
| 51 | Jimeng | - | - | ImgG | Jimeng Video、异步图像（配置） | 同步适配器面向图像；视频另走任务适配器和 `/jimeng/` 路由 |
| 52 | Vidu | - | - | - | Vidu Video | 支持文生、图生、首尾帧和参考图任务模式 |
| 53 | Submodel | Chat | - | - | - | 代码明确拒绝其他转换器 |
| 54 | DoubaoVideo | - | - | - | Doubao Video | 仅视频任务 |
| 55 | Sora | - | - | - | OpenAI Video | 支持创建、查询、内容代理和 remix |
| 56 | Replicate | - | - | ImgG、ImgE | 异步图像（配置） | 其他转换器明确未实现 |
| 57 | ChatGPT Subscription (Codex) | Resp、Compact、Alpha | - | - | - | 明确拒绝 Chat、Msg、Gemini、Embedding、Image、Audio 和 Rank |
| 58 | Advanced Custom | 按 route/converter 配置 | 按配置 | 按配置 | RT（按配置）、异步图像（配置） | 可覆盖已注册同步路由和 Alpha；无视频 TaskAdaptor |
| 59 | Sub2API | Chat、Comp、Msg、Resp、Compact、Gemini、Mod、Alpha | Emb | ImgG、ImgE | 异步图像（配置） | 继承 New API 适配器；Audio、Rank、RT 和 Video 不支持 |
| 60 | New API | Chat、Comp、Msg、Resp、Compact、Gemini、Mod、Alpha | Emb | ImgG、ImgE | 异步图像（配置） | 适配器明确拒绝 Audio 和 Rank；无 RT 和 Video |

## 隐藏或遗留渠道类型

这些类型存在于后端常量中，但当前前端 `CHANNEL_TYPES` 不提供选择。删除、恢复或改变它们的映射时仍需维护本节。

| ID | 渠道类型 | 当前状态 |
| ---: | --- | --- |
| 6 | OpenAIMax | 回落到 OpenAI 兼容适配器；前端隐藏 |
| 9 | AILS | 回落到 OpenAI 兼容适配器；前端隐藏 |
| 10 | AIProxy | 回落到 OpenAI 兼容适配器；前端隐藏 |
| 11 | PaLM | 仅 Chat，映射到 PaLM `generateMessage`；前端隐藏 |
| 12 | API2GPT | 回落到 OpenAI 兼容适配器；前端隐藏 |
| 13 | AIGC2D | 回落到 OpenAI 兼容适配器；前端隐藏 |
| 21 | AIProxyLibrary | `ChannelType2APIType` 有映射，但 `GetAdaptor` 没有对应适配器，当前不可用且前端隐藏 |

## 视频和异步任务端点

| 客户端入口 | 支持的渠道类型 | 说明 |
| --- | --- | --- |
| `POST /v1/videos` | OpenAI、Sora、Gemini、Vertex AI、Ali、Kling、Jimeng、Vidu、VolcEngine、DoubaoVideo、MiniMax | 统一视频创建入口；按渠道任务适配器转换 |
| `GET /v1/videos/:task_id` | 上述视频任务渠道 | 查询本系统任务并调用对应上游查询逻辑 |
| `POST /v1/videos/:video_id/remix` | OpenAI、Sora | 只有 Sora/OpenAI 任务适配器实现真正的 remix 上游路径 |
| `GET /v1/videos/:task_id/content` | 已完成且具有可代理内容的视频任务 | 公共内容能力 URL，由任务记录和上游结果决定 |
| `POST /v1/video/generations`、`GET /v1/video/generations/:task_id` | 与统一视频任务渠道相同 | 旧版兼容入口 |
| `/kling/v1/videos/text2video`、`/kling/v1/videos/image2video` 及查询路由 | Kling | Kling 官方格式兼容入口 |
| `POST /jimeng/` | Jimeng | 即梦官方格式入口 |
| `POST /v1/images/generations`，请求 `async=true` 或 Header `Prefer: respond-async` | 表中标记“异步图像（配置）”且已开启开关的渠道 | 按渠道选择 Ali、New API 或 Grsai 提交协议；协议与上游不匹配时直接报错，不降级同步 |
| `GET /v1/images/generations/:task_id` | 异步图像任务 | 查询本系统任务；提交时已快照上游轮询 URL 与认证配置 |
| `/mj/submit/*`、`/mj/task/*`、`/mj/insight-face/*` | MjProxy、MjProxyPlus | Midjourney 专用任务集合 |
| `/suno/submit/:action`、`/suno/fetch` | SunoAPI | Suno 专用任务集合 |

## 已注册但未实现的端点

以下路由会进入 `RelayNotImplemented`，不得在渠道矩阵中标记为支持：

- `/v1/images/variations`
- `/v1/files` 及文件详情/内容/删除路由
- `/v1/fine-tunes` 及详情、取消、事件路由
- `DELETE /v1/models/:model`

## 维护规则

每次新增、删除或修改渠道能力时，必须在同一变更中更新本文档。至少检查：

1. `constant/channel.go`、`constant/api_type.go` 和 `common/api_type.go` 的渠道/API 类型及映射。
2. `web/src/features/channels/constants.ts` 的前端可选类型和显示顺序。
3. `relay/relay_adaptor.go` 的同步适配器与任务适配器选择。
4. `router/relay-router.go`、`router/video-router.go` 及其他公开路由文件。
5. 对应适配器的请求转换、`GetRequestURL`、`DoRequest`、`DoResponse`，不能只凭某一个函数存在就标记支持。
6. 模型限定、API 版本、密钥格式、Base URL、流式、异步轮询和响应格式等条件限制。
7. 新渠道类型必须新增独立表格行；不支持的端点要明确保持未列出或在限制栏说明，禁止写成模糊的“兼容全部”。

源码入口：[`router/relay-router.go`](../router/relay-router.go)、[`router/video-router.go`](../router/video-router.go)、[`relay/relay_adaptor.go`](../relay/relay_adaptor.go)、[`common/api_type.go`](../common/api_type.go)、[`web/src/features/channels/constants.ts`](../web/src/features/channels/constants.ts)。
