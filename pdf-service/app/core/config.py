"""配置模块"""
import os
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """应用配置"""
    # 基础配置
    APP_ENV: str = "development"
    DEBUG: bool = True
    PORT: int = 8000

    # MongoDB (通过环境变量 MONGO_URI 设置)
    MONGO_URI: str = ""
    MONGO_DB: str = "kaoyan"

    # RabbitMQ (通过环境变量 RABBITMQ_URL 设置)
    RABBITMQ_URL: str = ""

    # Celery
    CELERY_BROKER_URL: str = ""
    CELERY_RESULT_BACKEND: str = "redis://localhost:6379/1"

    # MinIO
    MINIO_ENDPOINT: str = "192.168.0.200:9000"
    MINIO_ACCESS_KEY: str = "ivan"
    MINIO_SECRET_KEY: str = ""  # 通过环境变量 MINIO_SECRET_KEY 设置
    MINIO_BUCKET: str = "kaoyan-pdf"
    MINIO_SECURE: bool = False

    # LLM API (通过环境变量 LLM_API_KEY 或 DASHSCOPE_API_KEY 设置)
    DASHSCOPE_API_KEY: str = ""
    LLM_API_KEY: str = ""
    DASHSCOPE_BASE_URL: str = "https://api.minimaxi.com/v1"
    DASHSCOPE_MODEL: str = "MiniMax-M2.5"

    @property
    def llm_api_key(self) -> str:
        return self.LLM_API_KEY or self.DASHSCOPE_API_KEY

    # Go 服务回调地址
    GO_SERVER_URL: str = "http://localhost:8080"

    # CORS 允许的域名 (生产环境使用，逗号分隔)
    CORS_ALLOWED_ORIGINS: str = ""

    class Config:
        env_file = ".env"
        case_sensitive = True


settings = Settings()
