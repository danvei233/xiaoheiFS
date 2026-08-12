# update-sign — 更新清单签名工具（发布端）

签名端参考实现，产生 Flutter 客户端可验证的 Ed25519 更新清单字段。
签名契约见 `docs/update-signing-contract.md`。

## 用法

```bash
# 测试（本模块独立于根 go.work，需 GOWORK=off）
cd script/update-sign && GOWORK=off go test ./...

# 构建
cd script/update-sign && GOWORK=off go build -o update-sign .

# 签名一个未签名的清单 JSON
# 私钥只允许来自：UPDATE_SIGN_PRIVATE_KEY 环境 secret、0600 私钥文件、或 stdin。
UPDATE_SIGN_PRIVATE_KEY=<私钥hex> ./update-sign \
  -pkg com.example.app \
  -manifest manifest.json \
  -epoch 1 \
  -issued-at "$(date +%s)"

# 或者用 0600 私钥文件：
./update-sign -key-file /secure/path/update-signing.key \
  -pkg com.example.app -manifest manifest.json -epoch 1
```

## 私钥输入（安全约束）

**绝不通过 `-key`/argv 传私钥**（会泄漏到进程列表、shell history、CI 日志）。
工具只从下列**恰好一个**来源读取：

| 来源 | 参数/环境 | 说明 |
|---|---|---|
| 环境 secret | `UPDATE_SIGN_PRIVATE_KEY` | CI secret / 密钥管理服务注入 |
| 0600 文件 | `-key-file <path>` | 文件权限必须是 0600/0400；group/other 可读直接拒绝 |
| stdin | `-key-stdin` | 从管道读入，例如 `secret-manager get x \| update-sign -key-stdin ...` |

工具不打印私钥、不回显含密钥的输入。私钥可以是 32 字节 seed 或 64 字节完整私钥
（hex 或 base64）。`-epoch` 指定签名所用的 keyEpoch（客户端按清单里的 keyEpoch
选公钥，轮换见契约 §6.3）。`-issued-at` 缺省为当前时间。输出为带 `algorithm` 与
`signature` 的清单 JSON。

> 私钥只允许存在于发布维护者的密钥库 / CI secret。不得入库、不得随客户端发布。
> 客户端默认 epoch 1 的公钥必须由发布构建 `--dart-define=UPDATE_SIGN_PUBLIC_KEY`
> 注入，否则占位公钥 fail-closed（见契约 §6.1）。
