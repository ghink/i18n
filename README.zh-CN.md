# i18n

为 Go 提供的简易国际化方案。[English](README.md)

一个极小的国际化库，让你把译文**就地**保存在数据旁边。它不需要在运行时从外部
`.po`/`.json` 语言包读取文案，而是让每个字段自带一个多语言映射表，序列化和读取时再由库
根据当前 locale 挑选最合适的一条。

- 用一个简单的 `map[语言代码]文案` 存储译文。
- 通过 `context.Context` 携带用户的 locale（可带优先级列表）。
- 递归地把任意结构体、切片、映射、指针"剥"成单一语言。
- 底层依赖 [`golang.org/x/text/language`](https://pkg.go.dev/golang.org/x/text/language)
  做真正的最佳匹配协商，并提供确定性的兜底策略。

## 环境要求

- Go **1.27+**。

安装：

```sh
go get go.gh.ink/i18n
```

## 核心类型

```go
// I18n 是国际化的基础类型：
// key 是语言代码，value 是对应译文。
type I18n map[string]string
```

语言代码可以是任何 `golang.org/x/text/language` 能识别的标签，例如 `en`、`zh`、`zh-CN`、
`fr`、`pt-BR`。

```go
title := i18n.I18n{
    "en":    "Hello",
    "zh-CN": "你好",
    "fr":    "Bonjour",
}
```

locale 标签本身用 `Tag` / `Tags` 表示：

```go
type Tag language.Tag // 单个 BCP-47 语言标签
type Tags = []Tag     // 有序偏好列表，越靠前优先级越高
```

## 读取单个值：`Unwrap` / `Pick`

给定偏好列表，把一个映射表解析为一条记录：

```go
locales := i18n.MustParseLocales([]string{"zh-CN", "en"})

title.Unwrap(locales) // i18n.I18n{"zh-CN": "你好"} —— 仍是 I18n，只有一个 key
title.Pick(locales)   // "你好"                     —— 直接取字符串
```

`Unwrap` 返回一个**单键**的 `I18n`（若映射表为空或没有非空值则返回 `nil`），`Pick` 则直接
返回**字符串**。还有 `*Ctx` 变体，会从 context 中读取偏好列表：

```go
title.UnwrapCtx(ctx) // i18n.I18n
title.PickCtx(ctx)   // string
```

## 在 context 中携带 locale

locale 存放在 context 里，这样它能随请求一路传递到处理函数，并被序列化辅助函数读取：

```go
// 单个首选 locale。
ctx := i18n.WithLocale(context.Background(), i18n.MustParseLocale("zh-CN"))

// 完整偏好列表（越靠前越优先）。
ctx = i18n.WithLocales(ctx, i18n.MustParseLocales([]string{"zh-CN", "en"}))

// 读回来。
locales := i18n.GetLocales(ctx) // i18n.Tags
```

### 解析辅助函数

| 函数 | 行为 |
| --- | --- |
| `ParseLocale(string) (Tag, error)` | 解析单个 locale 字符串。 |
| `MustParseLocale(string) Tag` | 解析单个 locale，出错则 panic。 |
| `ParseLocales([]string) (Tags, error)` | 解析一组 locale。 |
| `MustParseLocales([]string) Tags` | 解析一组 locale，出错则 panic。 |
| `ParseAcceptLanguage(string) (Tags, error)` | 解析 HTTP `Accept-Language` 头，支持 `q` 权重。 |
| `MustParseAcceptLanguage(string) Tags` | 同上，出错则 panic。 |

在 HTTP 处理函数中，`ParseAcceptLanguage` 能把 `Accept-Language` 头转成偏好列表。它是
[`language.ParseAcceptLanguage`](https://pkg.go.dev/golang.org/x/text/language#ParseAcceptLanguage)
的薄封装：条目按 `q` 权重降序排列（缺失的 `q` 默认为 `1`），丢弃权重为 `0` 的条目，权重相同时
保持首次出现顺序。空或完全非法的头会返回 error，因此建议回退到默认 locale：

```go
locales, err := i18n.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
if err != nil || len(locales) == 0 {
    locales = i18n.MustParseLocales([]string{"en"}) // 合理的默认值
}
ctx := i18n.WithLocales(r.Context(), locales)
```

例如 `zh-CN,zh;q=0.9,en;q=0.8` 会得到优先级顺序 `zh-CN → zh → en`。

## 本地化整个结构体

用 `I18n` 字段声明数据，库会自动帮你遍历。它支持嵌套结构体、指针、切片、数组、映射和接口
—— 任何类型为 `I18n` 的导出字段都会被剥成最匹配的语言。

```go
type Product struct {
    Name   i18n.I18n   `json:"name"`
    Desc   i18n.I18n   `json:"desc"`
    Labels []i18n.I18n `json:"labels"`
}

p := Product{
    Name: i18n.I18n{"en": "Coffee", "zh-CN": "咖啡"},
    Desc: i18n.I18n{"en": "Hot drink", "zh-CN": "热饮"},
    Labels: []i18n.I18n{
        {"en": "Vegan", "zh-CN": "纯素"},
    },
}
```

有三个入口，全部由 `ctx` 中保存的 locales 驱动：

### `ToJSON(ctx, v) ([]byte, error)`

一步到位：把每个 `I18n` 字段剥成单一语言，再序列化为 JSON。由于单语言值会被编码成纯字符串
（见 [JSON 编码](#json-编码)），输出是易读文案而不是语言映射表：

```go
ctx := i18n.WithLocale(context.Background(), i18n.MustParseLocale("zh-CN"))
b, err := i18n.ToJSON(ctx, p)
// {"name":"咖啡","desc":"热饮","labels":["纯素"]}
```

### `Localise[Out](ctx, v) (Out, error)`

同样是剥 + 序列化，然后再解码到你指定的**另一个类型**。适合把携带 `I18n` 字段的模型转换成
纯字符串的扁平 DTO：

```go
type ProductDTO struct {
    Name   string   `json:"name"`
    Desc   string   `json:"desc"`
    Labels []string `json:"labels"`
}

dto, err := i18n.Localise[ProductDTO](ctx, p)
// ProductDTO{Name: "咖啡", Desc: "热饮", Labels: []string{"纯素"}}
```

### `LocaliseKeep[T](ctx, v T) T`

返回一个**同类型副本**，其中的 `I18n` 字段已被剥成单键。原始值保持不变，且字段仍是 `I18n`
（之后还能再次 `Pick`），而不是塌缩成 `string`：

```go
kept := i18n.LocaliseKeep(ctx, p) // 仍是 Product
kept.Name.Pick(nil)               // "咖啡" —— 已经收窄到单一语言
```

> `ToJSON` / `Localise` 会把 `I18n` 塌缩成字符串以便对外输出；`LocaliseKeep` 保留 `I18n`
> 类型，方便后续再次解析。三者都**不会修改入参**：它们在深拷贝上操作，原值始终不变。

## JSON 编码

`I18n` 实现了 `MarshalJSON` / `UnmarshalJSON`，能自然地往返序列化，编码结果取决于它当前持有
多少条记录：

| 值 | 编码结果 |
| --- | --- |
| `I18n{}`（空） | `""` |
| `I18n{"en": "Hello"}`（单条） | `"Hello"` |
| `I18n{"en": "Hi", "zh": "你好"}`（多条） | `{"en":"Hi","zh":"你好"}` |

解码与之对称：

| JSON | 解码结果 |
| --- | --- |
| `null` / 空 | `nil` |
| `"你好"`（字符串） | `I18n{"und": "你好"}`（`und` 表示未确定语种） |
| `{"en":"Hi"}`（对象） | `I18n{"en": "Hi"}` |

这正是本地化后的结构体能序列化成纯字符串的原因：剥完之后每个字段只剩一条记录。

## 匹配与兜底

在根据偏好列表解析映射表时，`i18n` 会：

1. 从**值非空且 key 是合法语言标签**的记录中构造候选集合（去除重复标签）。
2. 从请求的偏好列表中剔除 root / `und`（即 `Tag{}`）条目。
3. 使用 `language.NewMatcher(...).Match(preferences...)` 挑选最佳匹配 —— 从而获得真正的协商
   能力，比如请求 `zh-HK` 仍能回退到 `zh` 等。
4. 多个候选得分相同时（例如 `zh` 与 `zh-CN` 都满足 `zh` 请求），字典序最小的 key 胜出，结果
   永不依赖映射表遍历顺序。
5. 若完全匹配不上（`language.No`），则兜底选择**字典序最小且值非空的 key**，使结果不受映射表
   遍历顺序影响，保持确定性。

如果映射表为空或没有任何非空值，`Unwrap` 返回 `nil`，`Pick` 返回 `""`。

## 完整示例

```go
package main

import (
    "context"
    "fmt"

    "go.gh.ink/i18n"
)

type Greeting struct {
    Text i18n.I18n `json:"text"`
}

func main() {
    ctx := i18n.WithLocales(
        context.Background(),
        i18n.MustParseLocales([]string{"fr", "en"}),
    )

    g := Greeting{Text: i18n.I18n{"en": "Hello", "fr": "Bonjour"}}

    b, _ := i18n.ToJSON(ctx, g)
    fmt.Println(string(b)) // {"text":"Bonjour"}

    fmt.Println(g.Text.PickCtx(ctx)) // Bonjour
}
```

## API 一览

| 符号 | 说明 |
| --- | --- |
| `type I18n map[string]string` | 译文映射表。 |
| `type Tag` / `type Tags` | locale 标签 / 有序偏好列表。 |
| `WithLocale` / `WithLocales` / `GetLocales` | 在 context 中读写 locale。 |
| `ParseLocale` / `MustParseLocale` / `ParseLocales` / `MustParseLocales` | 解析 locale 字符串。 |
| `ParseAcceptLanguage` / `MustParseAcceptLanguage` | 解析 `Accept-Language` 头，支持 `q` 权重。 |
| `I18n.Unwrap` / `UnwrapCtx` | 解析为单键 `I18n`。 |
| `I18n.Pick` / `PickCtx` | 解析为纯 `string`。 |
| `ToJSON` | 剥语言 + 序列化 JSON。 |
| `Localise[Out]` | 剥语言 + 解码到目标类型。 |
| `LocaliseKeep[T]` | 剥语言并返回同类型副本。 |
| `Marshal` / `Unmarshal` | JSON 编解码器（`encoding/json/v2`）。 |

## 许可证

采用 [Apache License 2.0](LICENSE) 授权。
