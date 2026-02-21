"""知识点提取服务 - 调用 MiniMax LLM API"""
import json
import logging
from typing import Optional

from openai import OpenAI

from app.core.config import settings

logger = logging.getLogger(__name__)


class KnowledgeExtractor:
    """使用通义千问3从章节内容中提取知识点"""

    def __init__(self):
        if settings.DASHSCOPE_API_KEY and settings.DASHSCOPE_API_KEY != "mock":
            self.client = OpenAI(
                api_key=settings.DASHSCOPE_API_KEY,
                base_url=settings.DASHSCOPE_BASE_URL,
            )
        else:
            self.client = None
            logger.warning("DASHSCOPE_API_KEY 未配置，知识点提取将使用mock模式")

    def extract_knowledge_points(self, chapter_title: str, chapter_text: str, subject: str) -> list[dict]:
        """
        从章节内容中提取知识点

        Args:
            chapter_title: 章节标题
            chapter_text: 章节文本内容
            subject: 科目

        Returns:
            知识点列表
        """
        if not self.client:
            return self._mock_extract(chapter_title, subject)

        prompt = f"""请从以下考研教材章节内容中提取知识点。

科目: {subject}
章节: {chapter_title}

内容片段:
{chapter_text[:3000]}

请按以下JSON格式返回知识点列表:
{{
    "knowledge_points": [
        {{
            "name": "知识点名称",
            "difficulty": 3,
            "importance": 4,
            "estimated_hours": 2.0,
            "tags": ["标签1", "标签2"]
        }}
    ]
}}

要求:
1. difficulty 和 importance 范围 1-5
2. estimated_hours 为预估学习时间(小时)
3. 提取核心知识点，数量控制在3-10个
4. tags 标注知识点类型(基础/核心/难点/易错等)"""

        try:
            response = self.client.chat.completions.create(
                model=settings.DASHSCOPE_MODEL,
                messages=[
                    {"role": "system", "content": "你是一个考研辅导专家，擅长分析教材内容并提取核心知识点。请只返回JSON格式。"},
                    {"role": "user", "content": prompt},
                ],
                temperature=0.3,
                max_tokens=2048,
            )

            content = response.choices[0].message.content
            # 尝试解析JSON
            result = json.loads(content)
            return result.get("knowledge_points", [])

        except json.JSONDecodeError:
            logger.warning(f"AI返回的不是有效JSON: {chapter_title}")
            return self._mock_extract(chapter_title, subject)
        except Exception as e:
            logger.error(f"知识点提取失败: {e}")
            return self._mock_extract(chapter_title, subject)

    def generate_chapter_summary(self, chapter_title: str, chapter_text: str) -> str:
        """生成章节摘要"""
        if not self.client:
            return f"{chapter_title} 的内容摘要（mock模式）"

        try:
            response = self.client.chat.completions.create(
                model=settings.DASHSCOPE_MODEL,
                messages=[
                    {"role": "system", "content": "你是一个教材分析专家。请用2-3句话总结以下章节的核心内容。"},
                    {"role": "user", "content": f"章节: {chapter_title}\n\n内容:\n{chapter_text[:2000]}"},
                ],
                temperature=0.3,
                max_tokens=256,
            )
            return response.choices[0].message.content
        except Exception as e:
            logger.error(f"生成摘要失败: {e}")
            return ""

    def _mock_extract(self, chapter_title: str, subject: str) -> list[dict]:
        """Mock 模式：返回示例知识点"""
        subject_map = {
            "math": ["函数与极限", "导数与微分", "积分计算"],
            "english": ["阅读理解", "写作技巧", "长难句分析"],
            "politics": ["马克思主义原理", "毛泽东思想", "中国特色社会主义"],
            "professional": ["核心概念", "方法论", "案例分析"],
        }
        names = subject_map.get(subject, ["知识点A", "知识点B", "知识点C"])

        return [
            {
                "name": f"{chapter_title} - {name}",
                "difficulty": 3,
                "importance": 4,
                "estimated_hours": 2.0,
                "tags": ["基础", "核心概念"],
            }
            for name in names
        ]
