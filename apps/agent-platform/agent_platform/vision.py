from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True, slots=True)
class ImageInput:
    filename: str
    media_type: str
    data: bytes


@dataclass(frozen=True, slots=True)
class VisionResult:
    ocr_text: str = ""
    summary: str = ""


class LocalVisionClient:
    def analyze(self, text: str, images: list[ImageInput]) -> VisionResult:
        if not images:
            return VisionResult()
        names = ", ".join(item.filename or f"image-{idx + 1}" for idx, item in enumerate(images))
        return VisionResult(
            ocr_text=f"已收到 {len(images)} 张图片：{names}。当前未配置真实 Vision provider，仅作为附件证据保存。",
            summary="local vision placeholder",
        )
