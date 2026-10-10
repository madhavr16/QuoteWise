"""Reserved internal API boundary (Milestone 1 stub).

The Go API is the only intended caller (spec §9). Endpoints under
/internal/v1 will be implemented in Milestone 3+; until then every
method returns a consistent 501 so the reserved surface is documented
without pretending any feature exists.
"""

from fastapi import APIRouter
from fastapi.responses import JSONResponse

from app.server.errors import error_body

router = APIRouter(prefix="/internal/v1")

_MESSAGE = "This internal endpoint is not implemented yet (planned for Milestone 3+)."


@router.api_route("/{path:path}", methods=["GET", "POST", "PUT", "PATCH", "DELETE"])
def not_implemented() -> JSONResponse:
    return JSONResponse(
        status_code=501,
        content=error_body("not_implemented", _MESSAGE),
    )
