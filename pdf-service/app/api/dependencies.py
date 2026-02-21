"""API 依赖注入模块"""
import logging
import secrets
from fastapi import Security, HTTPException, status
from fastapi.security import APIKeyHeader

from app.core.config import settings

logger = logging.getLogger(__name__)

api_key_header = APIKeyHeader(name="X-API-Key", auto_error=False)

async def get_api_key(api_key: str = Security(api_key_header)):
    """验证内部 API Key"""
    if not settings.INTERNAL_API_KEY:
        logger.error("安全漏洞: 未配置 INTERNAL_API_KEY")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Server misconfiguration: missing INTERNAL_API_KEY"
        )
    
    if not api_key:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Missing API Key"
        )

    # 恒定时间比较，防止时序攻击
    if not secrets.compare_digest(api_key, settings.INTERNAL_API_KEY):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Invalid API Key"
        )
    return api_key
