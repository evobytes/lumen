import pytest
from fastapi.testclient import TestClient

from lumen.api import create_app
from lumen.backend import PresentationBackend
from lumen.model import Presentation, Slide


def client(token: str | None = None) -> TestClient:
    backend = PresentationBackend(Presentation(Slide(f"s{i}") for i in range(3)))
    return TestClient(create_app(backend, token))


def test_status_and_slides():
    c = client()
    s = c.get("/api/v1/status").json()
    assert (s["index"], s["count"], s["blanked"]) == (0, 3, False)
    assert [x["text"] for x in c.get("/api/v1/slides").json()] == ["s0", "s1", "s2"]


def test_navigation():
    c = client()
    assert c.post("/api/v1/next").json()["index"] == 1
    assert c.post("/api/v1/next").json()["index"] == 2
    assert c.post("/api/v1/next").json()["index"] == 2
    assert c.post("/api/v1/prev").json()["index"] == 1
    assert c.post("/api/v1/slides/0/show").json()["index"] == 0


def test_show_out_of_range_is_404():
    assert client().post("/api/v1/slides/9/show").status_code == 404


def test_blank_set_and_toggle():
    c = client()
    assert c.post("/api/v1/blank", json={"blank": True}).json()["blanked"] is True
    assert c.post("/api/v1/blank", json={}).json()["blanked"] is False  # toggle
    assert c.post("/api/v1/blank").json()["blanked"] is True  # no body also toggles


def test_replace_slides():
    c = client()
    c.post("/api/v1/next")
    r = c.put("/api/v1/slides", json={"slides": [{"text": "x", "title": "T"}]})
    assert r.json()["count"] == 1 and r.json()["index"] == 0
    assert c.get("/api/v1/slides").json()[0]["label"] == "T"


def test_output_and_screens():
    c = client()
    assert c.get("/api/v1/screens").json() == []
    assert c.put("/api/v1/output", json={"screen": None}).status_code == 200
    assert c.put("/api/v1/output", json={"screen": 3}).status_code == 422


def test_remote_page_served():
    r = client().get("/")
    assert r.status_code == 200 and "Lumen remote" in r.text


@pytest.mark.parametrize(
    "headers,ok",
    [
        ({}, False),
        ({"X-API-Token": "wrong"}, False),
        ({"X-API-Token": "secret"}, True),
        ({"Authorization": "Bearer secret"}, True),
        ({"Authorization": "Bearer nope"}, False),
    ],
)
def test_token_auth(headers, ok):
    c = client(token="secret")
    assert (c.get("/api/v1/status", headers=headers).status_code == 200) is ok
    assert c.get("/").status_code == 200  # the remote page itself is public
