"""PDF解析服务测试"""
import pytest


class TestPDFParser:
    """PDF文本提取测试"""

    def test_parser_import(self):
        """测试 PDFParser 可以正常导入"""
        from app.services.pdf_parser import PDFParser
        parser = PDFParser()
        assert parser is not None

    def test_extract_chapters_empty(self):
        """测试空解析结果的章节提取"""
        from app.services.pdf_parser import PDFParser
        parser = PDFParser()
        result = {
            "total_pages": 0,
            "pages": [],
            "toc": [],
            "has_text": False,
            "ocr_needed_pages": [],
        }
        chapters = parser.extract_chapters(result)
        assert chapters == []


class TestOCRService:
    """OCR服务测试"""

    def test_ocr_import(self):
        """测试 OCRService 可以正常导入"""
        from app.services.ocr_service import OCRService
        ocr = OCRService()
        assert ocr is not None


class TestKnowledgeExtractor:
    """知识点提取测试"""

    def test_mock_extract(self):
        """测试 mock 模式下的知识点提取"""
        from app.services.knowledge_extractor import KnowledgeExtractor
        extractor = KnowledgeExtractor()
        kps = extractor._mock_extract("第一章 极限", "math")
        assert len(kps) > 0
        assert "name" in kps[0]
        assert "difficulty" in kps[0]
        assert "importance" in kps[0]


class TestSchemas:
    """数据模型测试"""

    def test_parse_request(self):
        """测试ParseRequest模型"""
        from app.models.schemas import ParseRequest
        req = ParseRequest(
            textbook_id=1,
            user_id=1,
            file_key="test/file.pdf",
            subject="math",
            title="高等数学"
        )
        assert req.textbook_id == 1
        assert req.subject.value == "math"
