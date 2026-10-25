# Vinted 视频渠道适配

上游文档：https://vinted.cam/developers 。本适配只修改网关，画布继续使用现有视频入口和字段。

## 渠道配置与模型映射

OpenAI 或 Sora 渠道设置 `video_protocol=vinted`，Base URL 填上游基址。未显式设置协议且 Base URL 为 `vinted.cam` 或其子域的现有渠道也会识别。

客户端 `model` 是本地模型别名，按渠道模型映射得到上游 `model_id`。例如 `seedance2.0` 可映射为 `frameflow:seedance-2.0`。网关不猜测别名、不把所有模型固定到 Seedance，也不维护只允许旧名字的白名单；映射值必须存在于当前 `/v1/catalog` 且 `available=true`。不同模型的参数能力不通用。

## 保留的客户端请求

```http
POST /v1/videos
Authorization: Bearer <网关密钥>
Content-Type: application/json
Idempotency-Key: <独立操作编号>
```

```json
{
  "model": "seedance2.0",
  "prompt": "雨后城市街道，镜头缓缓向前推进",
  "duration": 10,
  "ratio": "9:16",
  "resolution": "720p",
  "image_urls": ["https://example.com/reference.png"]
}
```

无参考图时省略图片字段。继续兼容 `image/image_url/images/image_refs/image_urls`，并接受画布满血版的 `reference_images`，按既有顺序合并并去重；旧 `image_ids` 通过认证读取原图后换成新图库资产。时长还接受数字或整数字符串 `seconds`，画幅接受 `aspect_ratio` 或 `metadata.ratio`，画质接受 `metadata.resolution`；明确的 `duration/ratio/resolution` 优先，最终统一发送官网字段。可选 `input_mode` 只能使用目录声明的模式，例如 C4 的 `frames`；没有指定时根据参考图和目录选择 `text` 或 `references`。

## 内部转换

1. 在模型映射之后读取当前目录；拒绝未就绪、过期、未上架或不存在的模型。
2. 以目录的 `parameter_schema` 默认值和常量为基础，填入客户端明确提供的时长、画幅、画质。仅发送该模型声明的参数；未声明画质的模型省略 `resolution`。非法的已声明参数不会被改成其他值。时长还受网关统一安全上限约束，并同步到本地计费输入。
3. 按 `input_schema`、`x-canvas-inputs.image` 和组合约束校验提示词及参考图。网关提前校验当前目录使用的类型、必填、常量、枚举、上下界、长度、数量、`anyOf/allOf/if/then/else`；完整 Schema 和媒体内容仍由报价接口最终校验。
4. 使用网关现有 SSRF 防护下载器读取外部图片，不向图片站发送 API Key。图片仅接受原始 PNG/JPEG/WebP 字节，每张最多 10 MiB，不压缩或重编码。通过带稳定名称的 `POST /v1/files` 上传，要求响应含图片 `asset_id`，不能使用临时 `image_id` 替代。
5. 构造官网嵌套正文，调用 `POST /v1/videos/quote`；要求非空 `quote_id` 且目录版本一致。报价或资产响应不合法时停止，不调用生成。上游拒绝保留 HTTP 状态、公开错误码、消息和请求编号。
6. 创建时使用报价的同一份 JSON 字节，带 `Expected-Quote-Id` 和既有作用域幂等键。该执行内再次构建请求复用原素材、报价和正文。目录、上传、报价请求不跟随重定向，以免泄露上游凭据。

例如向上游发送：

```json
{
  "model_id": "frameflow:seedance-2.0",
  "catalog_revision": "当前目录 revision",
  "input_mode": "references",
  "input": {
    "prompt": "雨后城市街道，镜头缓缓向前推进",
    "images": [{"asset_id": "上传返回的资产编号", "kind": "image"}]
  },
  "parameters": {"duration": 10, "ratio": "9:16", "resolution": "720p"}
}
```

上游报价不改变网关原有售价配置、预扣和结算方式。这里的图片上传及报价不会发起视频生成；生成仍只有原来的创建接口。不要把请求超时当成未受理，也不要为重试换幂等键。网关继续使用已有持久化请求占位和渠道凭据绑定；跨请求重试时若目录变化，上游可能以同键不同正文拒绝，需核对原操作，禁止自动换键重下单。

## 查询与下载

公开任务 ID、上游任务 ID 隔离和轮询配置快照保持原有机制。查询返回 `succeeded` 但明确 `has_file=false` 时继续等待文件归档；旧任务没有该字段时兼容既有状态。

内容代理认证调用 `/v1/videos/{上游任务ID}/signed_url`，不添加旧 `download/passthrough` 查询参数，不要求当前合同未承诺的 `quality/expires_at` 字段；仍拒绝明确标记的预览文件。相对地址按上游基址解析，成片使用独立匿名请求，保留断点下载能力。协议不支持 remix，也不新增对外素材上传或音视频参考路由。

## 验证

单元测试使用本地模拟上游，覆盖映射后的新模型、动态默认值与计费时长、参数和组合限制、参考图原字节上传、旧图片别名、报价与创建一致性、错误传播、幂等键及下载响应。不使用真实渠道密钥，不调用付费生成。更新后需部署新的网关二进制，画布无需因本次报错修改或重新发布。
