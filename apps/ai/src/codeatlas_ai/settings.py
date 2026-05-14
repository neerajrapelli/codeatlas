from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="CODEATLAS_AI_",
        env_file=".env",
        env_file_encoding="utf-8",
        extra="ignore",
    )

    service_name: str = "codeatlas-ai"
    http_host: str = "0.0.0.0"
    http_port: int = 8001
    database_url: str | None = None
