"""
考研规划小程序 - PDF解析服务
FastAPI 应用入口
"""
import signal
import sys
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from app.api.routes import router as api_router
from app.core.config import settings

app = FastAPI(
    title="KaoYan PDF Service",
    description="考研规划小程序 PDF 解析服务",
    version="1.0.0",
)

# CORS 配置
# 生产环境应配置具体的允许域名 (通过 CORS_ALLOWED_ORIGINS 环境变量设置)
if settings.APP_ENV == "development":
    cors_origins = ["*"]
    cors_credentials = False
elif settings.CORS_ALLOWED_ORIGINS:
    cors_origins = [o.strip() for o in settings.CORS_ALLOWED_ORIGINS.split(",")]
    cors_credentials = True
else:
    cors_origins = []
    cors_credentials = False

app.add_middleware(
    CORSMiddleware,
    allow_origins=cors_origins,
    allow_credentials=cors_credentials,
    allow_methods=["*"],
    allow_headers=["*"],
)

# 注册路由
app.include_router(api_router, prefix="/api/v1")


@app.get("/health")
async def health_check():
    """健康检查"""
    return {"status": "healthy", "service": "pdf-service"}


if __name__ == "__main__":
    import uvicorn
    
    config = uvicorn.Config(
        "main:app",
        host="0.0.0.0",
        port=settings.PORT,
        reload=settings.DEBUG,
    )
    server = uvicorn.Server(config)
    
    def signal_handler(sig, frame):
        print("\n收到关闭信号，正在优雅关闭...")
        server.should_exit = True
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    server.run()
