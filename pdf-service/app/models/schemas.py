"""Pydantic 数据模型"""
from datetime import datetime
from enum import Enum
from typing import Optional
from pydantic import BaseModel, Field


class SubjectEnum(str, Enum):
    MATH = "math"
    ENGLISH = "english"
    POLITICS = "politics"
    PROFESSIONAL = "professional"


class ParseStatus(str, Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


# ========== Request Models ==========

class ParseRequest(BaseModel):
    """PDF解析请求"""
    textbook_id: int = Field(..., description="教材ID")
    user_id: int = Field(..., description="用户ID")
    file_key: str = Field(..., description="MinIO文件key")
    subject: SubjectEnum = Field(..., description="科目")
    title: str = Field("", description="教材标题")


class ParseCallbackRequest(BaseModel):
    """解析回调请求"""
    textbook_id: int
    mongo_doc_id: str
    total_chapters: int
    knowledge_points: int


# ========== Response Models ==========

class KnowledgePoint(BaseModel):
    """知识点"""
    name: str = Field(..., description="知识点名称")
    difficulty: int = Field(3, ge=1, le=5, description="难度1-5")
    importance: int = Field(3, ge=1, le=5, description="重要性1-5")
    estimated_hours: float = Field(1.0, description="预估学习时长(小时)")
    tags: list[str] = Field(default_factory=list, description="标签")


class Chapter(BaseModel):
    """章节"""
    chapter_no: int = Field(..., description="章节号")
    title: str = Field(..., description="章节标题")
    page_start: int = Field(0, description="起始页")
    page_end: int = Field(0, description="结束页")
    content_summary: str = Field("", description="内容摘要")
    knowledge_points: list[KnowledgePoint] = Field(default_factory=list)
    sub_chapters: list["Chapter"] = Field(default_factory=list)


class ParseResult(BaseModel):
    """解析结果"""
    textbook_id: int
    user_id: int
    file_key: str
    title: str
    subject: str
    total_pages: int = 0
    parse_method: str = "text"  # text / ocr / mixed
    chapters: list[Chapter] = Field(default_factory=list)
    metadata: dict = Field(default_factory=dict)
    created_at: datetime = Field(default_factory=datetime.now)
    updated_at: datetime = Field(default_factory=datetime.now)


class ParseStatusResponse(BaseModel):
    """解析状态响应"""
    textbook_id: int
    status: ParseStatus
    progress: float = 0.0
    message: str = ""
    result: Optional[ParseResult] = None


class HealthResponse(BaseModel):
    """健康检查响应"""
    status: str = "healthy"
    service: str = "pdf-service"
