"""PDF 解析服务 API 路由"""
import logging

from fastapi import APIRouter, HTTPException, Depends
from pymongo import MongoClient

from app.models.schemas import ParseRequest, ParseStatusResponse
from app.api.dependencies import get_api_key
from app.tasks.parse_task import parse_pdf_task
from app.core.config import settings

logger = logging.getLogger(__name__)

router = APIRouter(dependencies=[Depends(get_api_key)])

_mongo_client: MongoClient | None = None


def _get_mongo_db():
    global _mongo_client
    if _mongo_client is None:
        _mongo_client = MongoClient(settings.MONGO_URI)
    return _mongo_client[settings.MONGO_DB]


@router.post("/parse", summary="提交PDF解析任务")
async def submit_parse_task(req: ParseRequest):
    """
    提交PDF解析任务到消息队列

    流程: 接收请求 → 异步任务入队 → 返回任务ID
    """
    try:
        task = parse_pdf_task.delay(
            textbook_id=req.textbook_id,
            user_id=req.user_id,
            file_key=req.file_key,
            subject=req.subject.value,
            title=req.title,
        )
        logger.info(f"解析任务已提交: textbook_id={req.textbook_id}, task_id={task.id}")

        return {
            "code": 0,
            "message": "解析任务已提交",
            "data": {
                "task_id": task.id,
                "textbook_id": req.textbook_id,
                "status": "processing",
            },
        }
    except Exception as e:
        logger.error(f"提交解析任务失败: {e}")
        raise HTTPException(status_code=500, detail=f"提交解析任务失败: {str(e)}")


@router.get("/parse/{task_id}/status", summary="查询解析任务状态")
async def get_parse_status(task_id: str):
    """查询异步解析任务状态"""
    task = parse_pdf_task.AsyncResult(task_id)

    status_map = {
        "PENDING": "pending",
        "STARTED": "processing",
        "SUCCESS": "completed",
        "FAILURE": "failed",
        "RETRY": "processing",
    }

    result_data = None
    if task.state == "SUCCESS":
        result_data = task.result

    return {
        "code": 0,
        "message": "ok",
        "data": {
            "task_id": task_id,
            "status": status_map.get(task.state, "unknown"),
            "result": result_data,
        },
    }


@router.get("/parse/results/{textbook_id}", summary="获取解析结果")
async def get_parse_result(textbook_id: int):
    """从 MongoDB 获取 PDF 解析结果"""
    try:
        db = _get_mongo_db()
        result = db.pdf_parse_results.find_one(
            {"textbook_id": textbook_id},
            {"_id": 0},
        )

        if not result:
            raise HTTPException(status_code=404, detail="解析结果不存在")

        return {
            "code": 0,
            "message": "ok",
            "data": result,
        }
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"获取解析结果失败: {e}")
        raise HTTPException(status_code=500, detail=f"获取解析结果失败: {str(e)}")
