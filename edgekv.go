// 1. 实现 json schema 编辑
// 2. 中心化管理，分布式同步配置
// 3. Edge 边缘网关
//     1. Get 获取配置（本地存储）当服务器同步过来数据时，更新本地设置
//     2. Set 设置配置，设置的结构会先存储在本地，并同步到服务器上去。
//     3. Watch 监听来至远程的变化
//     4. Bind 绑定一个非持久化的数据，通常返回一个计算结果的值，无论是否在 Center 中心访问数据还是本地，这个值，可以被 Cache 在中心，也可以 Immediate 从 Edge 中读取
//     5.
// 4. Center 中心服务
//     1. Get 获取边缘节点数据，通常是缓存中获取，但也可以指定 Immediate 从 Edge 中读取
//     2. Set 设置配置，设置值会先存储在 Cache  中，并向 Edge 发送同步的消息队列，像数据同步到远端, 需要注意以下场景
//         1.  Edge 同步超时，如果消息投送超时，可以考虑持续化 轻型任务 (Light Job )  来后台执行尝试同步的任务
//     3. Watch  服务器也监听来至 Edge 的数据变化。
// 5. 结构化描述  参考 [https://rjsf-team.github.io/react-jsonschema-form/](https://rjsf-team.github.io/react-jsonschema-form/)
//     1. Edge ID 边缘节点，Center 上访问 配置需要指定 Edge ID 获取配置
//     2. Model 模型 配置 Root 上并不存储数据，必须要分成不同的 Model ，而 Schema 是为每个存储模型定义方案。
//     3. 编辑器依赖 JsonSchema , SchemaUI 的配置进行编辑，
//     4. 服务器支持 JsonSchema 的设置也读取
// 6. 同步通信
//     1.
// Package edgekv
package edgekv

type EdgeID string
