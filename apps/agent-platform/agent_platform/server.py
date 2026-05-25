from __future__ import annotations

import argparse
from contextlib import asynccontextmanager
from typing import Any

import uvicorn
from fastapi import BackgroundTasks, FastAPI, File, Form, HTTPException, Request, UploadFile
from fastapi.encoders import jsonable_encoder
from fastapi.responses import HTMLResponse, JSONResponse

from .chat_platform import ChatPlatformError, LarkImageDownloader, parse_chat_event
from .config import Config, VisionConfig, load_config
from .gateway import GatewayHTTPClient
from .repository import MySQLRepository, Repository
from .service import AgentPlatform
from .vision import ImageInput


def create_app(config: Config | None = None, repository: Repository | None = None, platform: AgentPlatform | None = None) -> FastAPI:
    @asynccontextmanager
    async def lifespan(app: FastAPI):
        if platform is not None:
            app.state.platform = platform
            yield
            return
        loaded = config or load_config()
        if repository is not None:
            repo = repository
        elif loaded.db_driver == "mysql":
            if loaded.mysql is None:
                raise RuntimeError("mysql config is required")
            repo = MySQLRepository(loaded.mysql)
        else:
            raise RuntimeError("Agent Platform production path requires DB_DRIVER=mysql")
        app.state.platform = AgentPlatform(
            loaded,
            repo,
            gateway=GatewayHTTPClient(
                loaded.gateway_endpoint,
                loaded.gateway_bearer_token,
                loaded.gateway_admin_bearer_token,
                loaded.max_investigation_seconds,
            ),
        )
        try:
            yield
        finally:
            app.state.platform.close()

    app = FastAPI(
        title="AI Troubleshooter Agent Platform",
        version="0.1.0",
        lifespan=lifespan,
        docs_url="/docs",
        redoc_url="/redoc",
    )

    @app.exception_handler(ValueError)
    def handle_value_error(_: Request, exc: ValueError) -> JSONResponse:
        return _json({"error": str(exc)}, 400)

    @app.exception_handler(KeyError)
    def handle_key_error(_: Request, exc: KeyError) -> JSONResponse:
        return _json({"error": str(exc).strip("'")}, 404)

    @app.exception_handler(RuntimeError)
    def handle_runtime_error(_: Request, exc: RuntimeError) -> JSONResponse:
        return _json({"error": str(exc)}, 502)

    @app.get("/healthz")
    def health(request: Request) -> JSONResponse:
        return _json(_platform(request).health())

    @app.get("/")
    @app.get("/web")
    def web(request: Request) -> HTMLResponse:
        cfg = _platform(request).config
        html = cfg.web_asset_path.read_text(encoding="utf-8")
        return HTMLResponse(html)

    @app.post("/api/v1/chat")
    @app.post("/web/api/chat")
    def chat(
        request: Request,
        background_tasks: BackgroundTasks,
        message: str = Form(""),
        async_: str = Form("0", alias="async"),
        case_no: str = Form(""),
        title: str = Form(""),
        images: list[UploadFile] = File(default=[]),
    ) -> JSONResponse:
        platform = _platform(request)
        image_inputs = _read_images(images, platform.config.vision)
        async_process = _truthy(async_)
        payload = platform.submit_chat(
            message=message,
            title=title,
            case_no=case_no,
            images=image_inputs,
            async_process=async_process,
        )
        if async_process:
            case_id = int(payload["case"]["id"])
            background_tasks.add_task(platform.process_case, case_id)
            return _json(payload, 202)
        return _json(payload)

    @app.get("/api/v1/overview")
    @app.get("/web/api/overview")
    def overview(request: Request) -> JSONResponse:
        return _json(_platform(request).overview())

    @app.get("/api/v1/cases/{case_ref:path}")
    @app.get("/web/api/cases/{case_ref:path}")
    def case_status(request: Request, case_ref: str) -> JSONResponse:
        return _json(_platform(request).get_case_payload(case_ref))

    @app.get("/api/v1/agent-runtimes")
    @app.get("/web/api/agent-runtimes")
    def list_agent_runtimes(request: Request) -> JSONResponse:
        return _json(_platform(request).list_agent_runtimes())

    @app.post("/api/v1/agent-runtimes/register")
    @app.post("/web/api/agent-runtimes/register")
    async def register_agent_runtime(request: Request) -> JSONResponse:
        return _json(_platform(request).register_agent_runtime(await request.json()), 201)

    @app.post("/api/v1/agent-runtimes/{runtime_id}/heartbeat")
    @app.post("/web/api/agent-runtimes/{runtime_id}/heartbeat")
    async def heartbeat_agent_runtime(request: Request, runtime_id: str) -> JSONResponse:
        return _json(_platform(request).heartbeat_agent_runtime(runtime_id, await request.json()))

    @app.patch("/api/v1/cases/{case_ref:path}")
    @app.put("/api/v1/cases/{case_ref:path}")
    @app.patch("/web/api/cases/{case_ref:path}")
    @app.put("/web/api/cases/{case_ref:path}")
    async def rename_case(request: Request, case_ref: str) -> JSONResponse:
        body = await request.json()
        return _json(_platform(request).rename_case(case_ref, str(body.get("title") or "")))

    @app.delete("/api/v1/cases/{case_ref:path}")
    @app.delete("/web/api/cases/{case_ref:path}")
    def delete_case(request: Request, case_ref: str) -> JSONResponse:
        return _json(_platform(request).delete_case(case_ref))

    @app.get("/api/v1/knowledge")
    @app.get("/web/api/knowledge")
    def list_knowledge(request: Request) -> JSONResponse:
        repo = _platform(request).repository
        return _json({"items": repo.list_knowledge(50)})

    @app.post("/api/v1/knowledge")
    @app.post("/web/api/knowledge")
    async def create_knowledge(request: Request) -> JSONResponse:
        body = await request.json()
        return _json(_platform(request).save_knowledge(_knowledge_payload(body)), 201)

    @app.get("/api/v1/knowledge/{knowledge_id}")
    @app.get("/web/api/knowledge/{knowledge_id}")
    def get_knowledge(request: Request, knowledge_id: int) -> JSONResponse:
        item = _platform(request).repository.get_knowledge(knowledge_id)
        if item is None:
            raise KeyError("knowledge item not found")
        return _json(item)

    @app.put("/api/v1/knowledge/{knowledge_id}")
    @app.patch("/api/v1/knowledge/{knowledge_id}")
    @app.put("/web/api/knowledge/{knowledge_id}")
    @app.patch("/web/api/knowledge/{knowledge_id}")
    async def update_knowledge(request: Request, knowledge_id: int) -> JSONResponse:
        body = await request.json()
        return _json(_platform(request).save_knowledge(_knowledge_payload(body), knowledge_id))

    @app.delete("/api/v1/knowledge/{knowledge_id}")
    @app.delete("/web/api/knowledge/{knowledge_id}")
    def delete_knowledge(request: Request, knowledge_id: int) -> JSONResponse:
        _platform(request).repository.delete_knowledge(knowledge_id)
        return _json({"deleted": True, "id": knowledge_id})

    @app.get("/api/v1/capabilities")
    @app.get("/web/api/capabilities")
    def capabilities(request: Request) -> JSONResponse:
        return _json({"items": _platform(request).repository.list_capabilities(200)})

    @app.post("/api/v1/capabilities/import")
    @app.post("/web/api/capabilities/import")
    async def import_tool_capabilities(request: Request) -> JSONResponse:
        body = await request.json()
        return _json(_platform(request).import_capabilities(body), 201)

    @app.post("/api/v1/capabilities/{capability_id}/publish")
    @app.post("/web/api/capabilities/{capability_id}/publish")
    def publish_capability(request: Request, capability_id: int) -> JSONResponse:
        return _json(_platform(request).publish_capability(capability_id))

    @app.post("/api/v1/capabilities/{capability_id}/disable")
    @app.post("/web/api/capabilities/{capability_id}/disable")
    def disable_capability(request: Request, capability_id: int) -> JSONResponse:
        return _json(_platform(request).disable_capability(capability_id))

    @app.post("/lark/events")
    @app.post("/feishu/events")
    async def chat_platform_event(request: Request, background_tasks: BackgroundTasks) -> JSONResponse:
        platform = _platform(request)
        try:
            event = parse_chat_event(await request.body(), platform.config.chat_platform)
        except ChatPlatformError as exc:
            raise HTTPException(status_code=exc.status_code, detail=str(exc)) from exc
        if event.challenge:
            return _json({"challenge": event.challenge})
        source = "feishu" if request.url.path.startswith("/feishu") else "lark"
        images, image_notes = LarkImageDownloader(platform.config.chat_platform).download_images(event)
        payload = platform.submit_channel_message(
            source=source,
            text=event.text,
            chat_id=event.chat_id,
            thread_id=event.thread_id,
            message_id=event.message_id,
            reporter_user_id=event.reporter_user_id,
            ocr_text="\n".join([event.ocr_text, *image_notes]),
            images=images,
            async_process=True,
        )
        if not payload.get("duplicate"):
            background_tasks.add_task(platform.process_case, int(payload["case"]["id"]))
        return _json(payload, 202)

    return app


def main() -> None:
    parser = argparse.ArgumentParser(description="Run the Python Agent Platform.")
    parser.add_argument("--host", default=None)
    parser.add_argument("--port", type=int, default=None)
    args = parser.parse_args()
    config = load_config()
    uvicorn.run(
        create_app(config=config),
        host=args.host or config.host,
        port=args.port or config.port,
        log_level="info",
        proxy_headers=True,
    )


def _platform(request: Request) -> AgentPlatform:
    return request.app.state.platform


def _json(payload: Any, status_code: int = 200) -> JSONResponse:
    return JSONResponse(status_code=status_code, content=jsonable_encoder(payload))


def _read_images(files: list[UploadFile], config: VisionConfig) -> list[ImageInput]:
    out: list[ImageInput] = []
    if len(files or []) > config.max_images_per_message:
        raise ValueError(f"too many images, max {config.max_images_per_message}")
    for item in files or []:
        data = item.file.read()
        if not data:
            continue
        if len(data) > config.max_image_bytes:
            raise ValueError(f"image {item.filename} exceeds max {config.max_image_bytes} bytes")
        media_type = item.content_type or "application/octet-stream"
        if not media_type.startswith("image/"):
            raise ValueError(f"file {item.filename} is not an image")
        out.append(ImageInput(filename=item.filename or "image", media_type=media_type, data=data))
    return out


def _truthy(value: str) -> bool:
    return str(value).strip().lower() in {"1", "true", "yes", "y", "on", "enabled"}


def _knowledge_payload(body: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": body.get("id") or 0,
        "title": body.get("title") or "",
        "issue_domain": body.get("issue_domain") or "",
        "issue_type": body.get("issue_type") or "",
        "typical_description": body.get("typical_description") or "",
        "recommended_steps": body.get("recommended_steps") or [],
        "common_causes": body.get("common_causes") or [],
        "useful_tools": body.get("useful_tools") or [],
        "confidence": body.get("confidence") or 0.7,
        "knowledge_status": "active",
        "observed_case_count": body.get("observed_case_count") or 1,
    }
