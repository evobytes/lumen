"""FastAPI application exposing remote control of the presentation."""

from __future__ import annotations

import secrets
from typing import Any

from fastapi import APIRouter, Depends, FastAPI, Header, HTTPException
from fastapi.responses import HTMLResponse
from pydantic import BaseModel

from lumen import __version__
from lumen.api.remote_ui import REMOTE_HTML
from lumen.backend import Backend
from lumen.model import Slide


class SlideIn(BaseModel):
    text: str
    title: str = ""


class SlidesIn(BaseModel):
    slides: list[SlideIn]


class BlankIn(BaseModel):
    blank: bool | None = None  # omitted/null toggles


class OutputIn(BaseModel):
    screen: int | None = None  # null selects the floating window


def create_app(backend: Backend, token: str | None = None) -> FastAPI:
    """Build the API. When ``token`` is set, /api/v1 requires it as a Bearer or X-API-Token."""

    def require_token(
        authorization: str | None = Header(default=None),
        x_api_token: str | None = Header(default=None),
    ) -> None:
        if not token:
            return
        supplied = x_api_token
        if supplied is None and authorization and authorization.lower().startswith("bearer "):
            supplied = authorization[7:]
        if supplied is None or not secrets.compare_digest(supplied, token):
            raise HTTPException(401, "missing or invalid API token")

    app = FastAPI(title="Lumen API", version=__version__)
    api = APIRouter(prefix="/api/v1", dependencies=[Depends(require_token)])

    @api.get("/status")
    def status() -> dict[str, Any]:
        return backend.status()

    @api.get("/slides")
    def list_slides() -> list[dict[str, Any]]:
        return backend.slides()

    @api.put("/slides")
    def replace_slides(body: SlidesIn) -> dict[str, Any]:
        return backend.set_slides([Slide(text=s.text, title=s.title) for s in body.slides])

    @api.post("/next")
    def next_slide() -> dict[str, Any]:
        return backend.next()

    @api.post("/prev")
    def prev_slide() -> dict[str, Any]:
        return backend.prev()

    @api.post("/slides/{index}/show")
    def show_slide(index: int) -> dict[str, Any]:
        try:
            return backend.goto(index)
        except IndexError as exc:
            raise HTTPException(404, str(exc)) from exc

    @api.post("/blank")
    def blank(body: BlankIn | None = None) -> dict[str, Any]:
        return backend.blank(body.blank if body else None)

    @api.get("/screens")
    def screens() -> list[dict[str, Any]]:
        return backend.screens()

    @api.put("/output")
    def set_output(body: OutputIn) -> dict[str, Any]:
        try:
            return backend.set_output(body.screen)
        except IndexError as exc:
            raise HTTPException(422, str(exc)) from exc

    app.include_router(api)

    @app.get("/", response_class=HTMLResponse, include_in_schema=False)
    def remote() -> str:
        return REMOTE_HTML

    return app
