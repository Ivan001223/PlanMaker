"""PDF 文本提取服务"""
import logging
from typing import Optional

import fitz  # PyMuPDF

logger = logging.getLogger(__name__)


class PDFParser:
    """PDF 文本提取器"""

    def extract_text(self, file_path: str) -> dict:
        """
        从PDF文件中提取文本和结构信息

        Args:
            file_path: PDF文件路径

        Returns:
            包含页面文本和元信息的字典
        """
        doc = fitz.open(file_path)
        result = {
            "total_pages": len(doc),
            "pages": [],
            "toc": [],
            "has_text": True,
            "ocr_needed_pages": [],
        }

        # 提取目录结构
        toc = doc.get_toc()
        result["toc"] = [
            {"level": level, "title": title, "page": page}
            for level, title, page in toc
        ]

        # 逐页提取文本
        for page_num in range(len(doc)):
            page = doc[page_num]
            text = page.get_text("text")

            page_data = {
                "page_num": page_num + 1,
                "text": text.strip(),
                "char_count": len(text.strip()),
                "has_text": len(text.strip()) > 10,
            }
            result["pages"].append(page_data)

            # 标记需要OCR的页面
            if not page_data["has_text"]:
                result["ocr_needed_pages"].append(page_num + 1)

        # 判断是否整体需要OCR
        text_pages = sum(1 for p in result["pages"] if p["has_text"])
        result["has_text"] = text_pages > len(doc) * 0.5

        doc.close()
        return result

    def extract_chapters(self, parse_result: dict) -> list[dict]:
        """
        从解析结果中提取章节结构

        Args:
            parse_result: extract_text 的返回结果

        Returns:
            章节列表
        """
        chapters = []
        toc = parse_result.get("toc", [])

        if toc:
            # 有目录结构，直接使用
            chapter_no = 0
            for i, item in enumerate(toc):
                if item["level"] == 1:
                    chapter_no += 1
                    end_page = toc[i + 1]["page"] - 1 if i + 1 < len(toc) else parse_result["total_pages"]

                    # 收集章节文本
                    chapter_text = ""
                    for page in parse_result["pages"]:
                        if item["page"] <= page["page_num"] <= end_page:
                            chapter_text += page["text"] + "\n"

                    chapters.append({
                        "chapter_no": chapter_no,
                        "title": item["title"],
                        "page_start": item["page"],
                        "page_end": end_page,
                        "content_text": chapter_text[:5000],  # 限制长度
                        "content_summary": "",
                        "knowledge_points": [],
                    })
        else:
            # 没有目录，按固定页数分章节
            pages_per_chapter = max(20, parse_result["total_pages"] // 10)
            chapter_no = 0
            for start_page in range(0, parse_result["total_pages"], pages_per_chapter):
                chapter_no += 1
                end_page = min(start_page + pages_per_chapter, parse_result["total_pages"])

                chapter_text = ""
                for page in parse_result["pages"]:
                    if start_page + 1 <= page["page_num"] <= end_page:
                        chapter_text += page["text"] + "\n"

                # 从章节文本中提取标题
                first_line = chapter_text.strip().split("\n")[0] if chapter_text.strip() else f"第{chapter_no}章"

                chapters.append({
                    "chapter_no": chapter_no,
                    "title": first_line[:50],
                    "page_start": start_page + 1,
                    "page_end": end_page,
                    "content_text": chapter_text[:5000],
                    "content_summary": "",
                    "knowledge_points": [],
                })

        return chapters
