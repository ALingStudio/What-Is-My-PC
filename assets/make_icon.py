#!/usr/bin/env python3
"""生成 What Is My PC 应用图标（清新蓝渐变圆角方块 + 白色显示器 + 问号）。
输出 assets/icon.ico（多尺寸，含 256 PNG 压缩条目）。"""

from PIL import Image, ImageDraw, ImageFont
import os

S = 256
HERE = os.path.dirname(os.path.abspath(__file__))

img = Image.new("RGBA", (S, S), (0, 0, 0, 0))

# 竖向渐变背景
top = (56, 189, 248)      # #38bdf8
bottom = (47, 124, 246)   # #2f7cf6
grad = Image.new("RGB", (1, S))
for y in range(S):
    t = y / (S - 1)
    grad.putpixel((0, y), tuple(int(top[i] + (bottom[i] - top[i]) * t) for i in range(3)))
grad = grad.resize((S, S))

mask = Image.new("L", (S, S), 0)
ImageDraw.Draw(mask).rounded_rectangle([0, 0, S - 1, S - 1], radius=58, fill=255)
img.paste(grad, (0, 0), mask)

d = ImageDraw.Draw(img)

# 显示器轮廓（白色）
d.rounded_rectangle([52, 58, 204, 162], radius=18, fill=(255, 255, 255, 255))
d.rectangle([118, 162, 138, 184], fill=(255, 255, 255, 255))
d.rounded_rectangle([96, 184, 160, 198], radius=7, fill=(255, 255, 255, 255))

# 问号
font = ImageFont.truetype("/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf", 92)
d.text((128, 112), "?", font=font, fill=(47, 124, 246, 255), anchor="mm")

sizes = [16, 20, 24, 32, 40, 48, 64, 128, 256]
img.save(os.path.join(HERE, "icon.ico"), sizes=[(s, s) for s in sizes])

# 同时导出 PNG 供预览
img.save(os.path.join(HERE, "icon.png"))
print("icon.ico / icon.png generated")
