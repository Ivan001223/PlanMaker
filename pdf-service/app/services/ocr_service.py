"""OCR 识别服务"""
import logging
from typing import Optional

logger = logging.getLogger(__name__)


class OCRService:
    """PaddleOCR 扫描版 PDF 识别服务"""

    def __init__(self):
        self._ocr = None

    def _init_ocr(self):
        """延迟初始化 PaddleOCR (加载模型较慢)"""
        if self._ocr is None:
            try:
                from paddleocr import PaddleOCR
                self._ocr = PaddleOCR(
                    use_angle_cls=True,
                    lang="ch",
                    use_gpu=False,  # 开发环境用CPU
                    show_log=False,
                )
                logger.info("PaddleOCR 初始化成功")
            except ImportError:
                logger.warning("PaddleOCR 未安装，OCR功能不可用")
                self._ocr = None
            except Exception as e:
                logger.error(f"PaddleOCR 初始化失败: {e}")
                self._ocr = None

    def ocr_page(self, image_path: str) -> str:
        """
        对单页图像进行 OCR 识别

        Args:
            image_path: 页面图片路径

        Returns:
            识别出的文本
        """
        self._init_ocr()
        if self._ocr is None:
            return "[OCR不可用] 请安装 PaddleOCR"

        try:
            result = self._ocr.ocr(image_path, cls=True)
            if not result or not result[0]:
                return ""

            lines = []
            for line in result[0]:
                text = line[1][0]
                confidence = line[1][1]
                if confidence > 0.5:
                    lines.append(text)

            return "\n".join(lines)
        except Exception as e:
            logger.error(f"OCR识别失败: {e}")
            return ""

    def ocr_pdf_pages(self, pdf_path: str, page_numbers: list[int]) -> dict[int, str]:
        """
        对PDF中指定页面进行 OCR 识别

        Args:
            pdf_path: PDF文件路径
            page_numbers: 需要OCR的页码列表 (1-indexed)

        Returns:
            {页码: 识别文本}
        """
        import fitz
        import uuid
        results = {}

        doc = fitz.open(pdf_path)
        for page_num in page_numbers:
            if page_num < 1 or page_num > len(doc):
                continue

            page = doc[page_num - 1]
            pix = page.get_pixmap(dpi=300)
            img_path = f"/tmp/ocr_{uuid.uuid4().hex}_{page_num}.png"
            pix.save(img_path)

            text = self.ocr_page(img_path)
            results[page_num] = text

            import os
            try:
                os.remove(img_path)
            except Exception:
                pass

        doc.close()
        return results
