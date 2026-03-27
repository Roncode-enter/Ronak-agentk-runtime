import os
import secrets
import logging

logger = logging.getLogger("agentk.config")

# --- JWT Configuration ---
# SECURITY: JWT_SECRET MUST be set via environment variable in production.
# A random fallback is generated for development only.
_default_jwt = secrets.token_urlsafe(48)
JWT_SECRET = os.getenv("JWT_SECRET", _default_jwt)
JWT_ALGORITHM = "HS256"
JWT_EXPIRY_HOURS = int(os.getenv("JWT_EXPIRY_HOURS", "24"))

if "JWT_SECRET" not in os.environ:
    logger.warning(
        "JWT_SECRET not set — using auto-generated secret. "
        "Sessions will NOT survive restarts. Set JWT_SECRET in production!"
    )

# --- Admin Credentials ---
# SECURITY: These MUST be set via environment variables in production.
ADMIN_EMAIL = os.getenv("ADMIN_EMAIL", "admin@agentk.ai")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "")

if not os.getenv("ADMIN_PASSWORD"):
    # Generate a random password for first run and print it
    ADMIN_PASSWORD = secrets.token_urlsafe(16)
    logger.warning(
        "ADMIN_PASSWORD not set — auto-generated password: %s  "
        "Set ADMIN_PASSWORD env var in production!", ADMIN_PASSWORD
    )

# --- Kubernetes ---
KUBECONFIG_PATH = os.getenv("KUBECONFIG", None)

# --- Premium features ---
PREMIUM_ENABLED = os.getenv("PREMIUM_ENABLED", "false").lower() == "true"

# --- CORS ---
CORS_ORIGINS = os.getenv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173").split(",")
