"""PDF 异步解析任务"""
import logging
import os
import tempfile
import time
from datetime import datetime

import httpx
from pymongo import MongoClient
from minio import Minio

from app.core.celery_app import celery_app
from app.core.config import settings
from app.services.pdf_parser import PDFParser
from app.services.ocr_service import OCRService
from app.services.knowledge_extractor import KnowledgeExtractor

logger = logging.getLogger(__name__)


@celery_app.task(bind=True, max_retries=3, default_retry_delay=60)
def parse_pdf_task(self, textbook_id: int, user_id: int, file_key: str, subject: str, title: str):
    """
    异步 PDF 解析任务

    流程:
    1. 从 MinIO 下载 PDF
    2. 使用 PyMuPDF 提取文本
    3. 对无文本页面使用 OCR
    4. 提取章节结构
    5. 使用 AI 提取知识点
    6. 存储结果到 MongoDB
    7. 回调 Go 服务更新状态
    """
    start_time = time.time()
    logger.info(f"开始解析 PDF: textbook_id={textbook_id}, file_key={file_key}")

    try:
        # 1. 从 MinIO 下载 PDF
        minio_client = Minio(
            settings.MINIO_ENDPOINT,
            access_key=settings.MINIO_ACCESS_KEY,
            secret_key=settings.MINIO_SECRET_KEY,
            secure=settings.MINIO_SECURE,
        )

        with tempfile.NamedTemporaryFile(suffix=".pdf", delete=False) as tmp_file:
            tmp_path = tmp_file.name
            minio_client.fget_object(settings.MINIO_BUCKET, file_key, tmp_path)
            logger.info(f"PDF 下载完成: {tmp_path}")

        # 2. 文本提取
        parser = PDFParser()
        parse_result = parser.extract_text(tmp_path)
        logger.info(f"文本提取完成: {parse_result['total_pages']}页, "
                     f"需OCR页面: {len(parse_result['ocr_needed_pages'])}")

        # 3. OCR 处理 (如果需要)
        parse_method = "text"
        if parse_result["ocr_needed_pages"]:
            ocr_service = OCRService()
            ocr_results = ocr_service.ocr_pdf_pages(tmp_path, parse_result["ocr_needed_pages"])

            # 将 OCR 结果合并回 pages
            for page_num, ocr_text in ocr_results.items():
                for page in parse_result["pages"]:
                    if page["page_num"] == page_num:
                        page["text"] = ocr_text
                        page["has_text"] = bool(ocr_text)
                        break

            if len(parse_result["ocr_needed_pages"]) == parse_result["total_pages"]:
                parse_method = "ocr"
            else:
                parse_method = "mixed"

        # 4. 提取章节结构
        chapters = parser.extract_chapters(parse_result)
        logger.info(f"章节提取完成: {len(chapters)}个章节")

        # 5. AI 知识点提取
        extractor = KnowledgeExtractor()
        total_knowledge_points = 0
        for chapter in chapters:
            kps = extractor.extract_knowledge_points(
                chapter["title"],
                chapter.get("content_text", ""),
                subject,
            )
            chapter["knowledge_points"] = kps
            total_knowledge_points += len(kps)

            # 生成摘要
            chapter["content_summary"] = extractor.generate_chapter_summary(
                chapter["title"],
                chapter.get("content_text", ""),
            )

            # 清除原始文本 (不存储到MongoDB)
            chapter.pop("content_text", None)

        logger.info(f"知识点提取完成: {total_knowledge_points}个知识点")

        # 6. 存储到 MongoDB
        mongo_client = MongoClient(settings.MONGO_URI)
        db = mongo_client[settings.MONGO_DB]

        duration_ms = int((time.time() - start_time) * 1000)
        doc = {
            "textbook_id": textbook_id,
            "user_id": user_id,
            "file_key": file_key,
            "title": title,
            "subject": subject,
            "total_pages": parse_result["total_pages"],
            "parse_method": parse_method,
            "chapters": chapters,
            "metadata": {
                "parse_duration_ms": duration_ms,
                "total_characters": sum(p["char_count"] for p in parse_result["pages"]),
                "ocr_pages": len(parse_result["ocr_needed_pages"]),
            },
            "created_at": datetime.now(),
            "updated_at": datetime.now(),
        }

        # Upsert
        result = db.pdf_parse_results.update_one(
            {"textbook_id": textbook_id},
            {"$set": doc},
            upsert=True,
        )
        mongo_doc_id = str(result.upserted_id) if result.upserted_id else ""
        if not mongo_doc_id:
            existing = db.pdf_parse_results.find_one({"textbook_id": textbook_id})
            mongo_doc_id = str(existing["_id"]) if existing else ""

        mongo_client.close()
        logger.info(f"MongoDB 存储完成: {mongo_doc_id}")

        # 7. 回调 Go 服务
        try:
            callback_url = f"{settings.GO_SERVER_URL}/api/v1/internal/textbooks/parse-callback"
            httpx.post(callback_url, json={
                "textbook_id": textbook_id,
                "mongo_doc_id": mongo_doc_id,
                "total_chapters": len(chapters),
                "knowledge_points": total_knowledge_points,
            }, timeout=10)
            logger.info("回调Go服务成功")
        except Exception as e:
            logger.warning(f"回调Go服务失败 (非致命): {e}")

        # 清理临时文件
        os.unlink(tmp_path)

        logger.info(f"PDF解析完成: textbook_id={textbook_id}, 耗时{duration_ms}ms")
        return {
            "status": "completed",
            "textbook_id": textbook_id,
            "chapters": len(chapters),
            "knowledge_points": total_knowledge_points,
            "duration_ms": duration_ms,
        }

    except Exception as e:
        logger.error(f"PDF解析失败: {e}", exc_info=True)
        # 清理
        if 'tmp_path' in locals():
            try:
                os.unlink(tmp_path)
            except Exception:
                pass
        raise self.retry(exc=e)
