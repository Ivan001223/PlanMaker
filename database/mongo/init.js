// =============================================
// 考研规划小程序 - MongoDB 初始化
// =============================================

// 切换到 kaoyan 数据库
db = db.getSiblingDB('kaoyan');

// -------------------------------------------
// PDF 解析结果集合
// -------------------------------------------
db.createCollection('pdf_parse_results');

db.pdf_parse_results.createIndex({ "textbook_id": 1 }, { unique: true });
db.pdf_parse_results.createIndex({ "user_id": 1 });
db.pdf_parse_results.createIndex({ "created_at": 1 });

// 插入示例文档结构 (会被真实数据覆盖)
db.pdf_parse_results.insertOne({
    "_id": ObjectId(),
    "textbook_id": 0,
    "user_id": 0,
    "file_key": "example/textbook.pdf",
    "title": "示例教材",
    "subject": "math",
    "total_pages": 0,
    "parse_method": "text",  // "text" | "ocr" | "mixed"
    "chapters": [
        {
            "chapter_no": 1,
            "title": "第一章 示例",
            "page_start": 1,
            "page_end": 20,
            "content_summary": "",
            "knowledge_points": [
                {
                    "name": "示例知识点",
                    "difficulty": 3,       // 1-5 难度
                    "importance": 4,       // 1-5 重要性
                    "estimated_hours": 2,  // 预估学习时长
                    "tags": ["基础", "核心概念"]
                }
            ],
            "sub_chapters": []
        }
    ],
    "metadata": {
        "parse_duration_ms": 0,
        "total_characters": 0,
        "ocr_pages": 0
    },
    "created_at": new Date(),
    "updated_at": new Date(),
    "_example": true
});

// 清理示例数据
db.pdf_parse_results.deleteMany({ "_example": true });

print("MongoDB 初始化完成: kaoyan 数据库已创建");
