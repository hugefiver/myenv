# Hyprland 模糊与透明度调参指南

## 设计目标

模拟 Windows Fluent Design 的 **Acrylic 亚克力材质**效果：毛玻璃透明 + 抗色阶，同时保持背景可辨识。

## Windows Acrylic 渲染管线

Acrylic 由 5 层从下到上合成：

```
┌─────────────────────────────┐
│  5. Noise Texture (2-4%)    │  ← 抗色阶（dithering）
├─────────────────────────────┤
│  4. Tint Color (60-70%)     │  ← 着色层，给表面"身份感"
├─────────────────────────────┤
│  3. Exclusion Blend         │  ← 保持 UI 元素对比度
├─────────────────────────────┤
│  2. Gaussian Blur (~30px)   │  ← 毛玻璃模糊
├─────────────────────────────┤
│  1. Background              │  ← 桌面壁纸 / 后方窗口内容
└─────────────────────────────┘
```

核心思路：**适度模糊 + 保持亮度 + 极微量噪点**。

## 色阶（Banding）原理

- **根因**：8-bit per channel = 每通道仅 256 级灰度
- 高斯模糊在两色之间插值时，中间值落在离散级别之间，产生可见"阶梯"
- **暗色渐变最严重**——人眼对暗部感知分辨率更高（Weber-Fechner 定律）
- Noise 的作用是 **有序抖动（ordered dithering）**，随机化量化误差，让大脑感知到更平滑的渐变

## Hyprland 参数说明

```ini
decoration {
    blur {
        size = 8          # 模糊半径，越大越糊。8 是保留背景辨识度的甜点
        passes = 3        # 模糊迭代次数，3 次足够柔和且不丢失细节
        noise = 0.01      # 噪点强度（0-1），0.01 刚好消除色阶但肉眼不可见
        contrast = 0.92   # 对比度系数（<1 降低），保持接近原始值避免变灰
        brightness = 0.92 # 亮度系数（<1 压暗），不过度压暗以减少暗部色阶
        vibrancy = 0.15   # 色彩鲜艳度提升，类似 Acrylic 的 saturation boost
        xray = true       # 穿透浮动窗口看到桌面背景（而非被遮挡窗口）
    }
}
```

## WezTerm 透明度

```lua
window_background_opacity = 0.65  -- 0.0 全透明，1.0 不透明
```

WezTerm 在 Wayland 上不做自己的模糊，透明度交给 compositor（Hyprland）处理。

## 参数调优逻辑

| 参数 | 调低的效果 | 调高的效果 | 注意事项 |
|------|-----------|-----------|---------|
| `size` | 模糊弱，背景清晰 | 模糊强，背景消失 | >12 时背景几乎不可辨 |
| `passes` | 模糊粗糙有颗粒感 | 模糊均匀柔和 | >4 性能下降且收益递减 |
| `noise` | 色阶可能可见 | 背景被噪点覆盖 | 0.01-0.02 是甜点 |
| `contrast` | 画面发灰 | 对比过强 | <0.85 背景变成一团灰 |
| `brightness` | 背景变暗+暗部色阶加重 | 背景过亮刺眼 | <0.85 暗部色阶明显 |
| `vibrancy` | 色彩平淡 | 色彩过艳 | 0.1-0.2 自然 |
| `opacity` | 更透明 | 更不透明 | <0.5 文字可读性下降 |

## 常见问题排查

**仍有色阶**：`noise` 从 0.01 调到 0.015，不要超过 0.02

**背景看不清**：降 `size`（8→6）或降 `passes`（3→2）

**整体太暗**：提高 `brightness`（0.92→0.95）

**颜色太灰**：提高 `contrast`（0.92→0.95）和 `vibrancy`（0.15→0.20）

**透明度不够 / 太透**：调 WezTerm 的 `window_background_opacity`

## 参考资料

- [Microsoft Acrylic Material](https://learn.microsoft.com/en-us/windows/apps/design/style/acrylic)
- [picom PR #952 — Bayer dithering for blur banding](https://github.com/yshui/picom/pull/952)
- [Hyprland Wiki — Configuring](https://wiki.hyprland.org/Configuring/)
