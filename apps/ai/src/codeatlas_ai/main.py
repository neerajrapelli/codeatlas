from __future__ import annotations

from fastapi import FastAPI

from codeatlas_ai import __version__
from codeatlas_ai.settings import Settings

settings = Settings()

app = FastAPI(title=settings.service_name, version=__version__)


@app.get("/health")
def health() -> dict[str, str]:
    return {
        "service": settings.service_name,
        "status": "ok",
        "version": __version__,
    }
