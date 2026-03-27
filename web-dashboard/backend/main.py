"""AgentK Dashboard — Managed Tier API Server.

Serves the React frontend and provides REST APIs for managing agents,
viewing costs, attestation reports, and simulation previews.
All state lives in Kubernetes — no database needed.
"""
from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import FileResponse
import os

from backend.routers import auth, agents, cost, attestation, simulation

app = FastAPI(
    title="AgentK Dashboard",
    description="Managed Tier dashboard for the AgentK Sovereign Agent Runtime",
    version="1.0.0",
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Register API routers
app.include_router(auth.router, prefix="/api/auth", tags=["Authentication"])
app.include_router(agents.router, prefix="/api/agents", tags=["Agents"])
app.include_router(cost.router, prefix="/api/cost", tags=["Cost Intelligence"])
app.include_router(attestation.router, prefix="/api/attestation", tags=["Attestation"])
app.include_router(simulation.router, prefix="/api/simulation", tags=["Simulation"])


@app.get("/api/health")
def health():
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
