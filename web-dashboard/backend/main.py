"""AgentK Dashboard — Managed Tier API Server.

Serves the React frontend and provides REST APIs for managing agents,
viewing costs, attestation reports, and simulation previews.
All state lives in Kubernetes — no database needed.
"""
from fastapi import FastAPI, Request
from fastapi.staticfiles import StaticFiles
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse, JSONResponse
import os
import logging
import time

from backend.routers import auth, agents, cost, attestation, simulation
from backend.config import CORS_ORIGINS

# --- Logging ---
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("agentk.dashboard")

app = FastAPI(
    title="AgentK Dashboard",
    description="Managed Tier dashboard for the AgentK Sovereign Agent Runtime",
    version="1.0.0",
)

# --- CORS (restricted origins) ---
app.add_middleware(
    CORSMiddleware,
    allow_origins=CORS_ORIGINS,
    allow_credentials=False,
    allow_methods=["GET", "POST", "DELETE"],
    allow_headers=["Authorization", "Content-Type"],
)


# --- Request logging middleware ---
@app.middleware("http")
async def log_requests(request: Request, call_next):
    start = time.time()
    response = await call_next(request)
    duration = round((time.time() - start) * 1000, 1)
    logger.info("%s %s -> %d (%sms)", request.method, request.url.path, response.status_code, duration)
    return response


# --- Global error handler ---
@app.exception_handler(Exception)
async def global_exception_handler(request: Request, exc: Exception):
    logger.error("Unhandled error on %s %s: %s", request.method, request.url.path, exc, exc_info=True)
    return JSONResponse(status_code=500, content={"detail": "Internal server error"})


# Register API routers
app.include_router(auth.router, prefix="/api/auth", tags=["Authentication"])
app.include_router(agents.router, prefix="/api/agents", tags=["Agents"])
app.include_router(cost.router, prefix="/api/cost", tags=["Cost Intelligence"])
app.include_router(attestation.router, prefix="/api/attestation", tags=["Attestation"])
app.include_router(simulation.router, prefix="/api/simulation", tags=["Simulation"])


@app.get("/api/health")
def health():
    """Health check — returns version and basic status."""
    return {"status": "healthy", "version": "1.0.0"}


# Serve React frontend (built static files)
STATIC_DIR = os.path.join(os.path.dirname(__file__), "..", "frontend", "dist")
if os.path.isdir(STATIC_DIR):
    app.mount("/assets", StaticFiles(directory=os.path.join(STATIC_DIR, "assets")), name="assets")

    @app.get("/{full_path:path}")
    async def serve_spa(full_path: str):
        """Serve the React SPA for all non-API routes."""
        file_path = os.path.join(STATIC_DIR, full_path)
        if os.path.isfile(file_path):
            return FileResponse(file_path)
        return FileResponse(os.path.join(STATIC_DIR, "index.html"))
