import os

JWT_SECRET = os.getenv("JWT_SECRET", "agentk-dashboard-secret-change-in-production")
JWT_ALGORITHM = "HS256"
JWT_EXPIRY_HOURS = 24

# Default admin credentials (change in production)
ADMIN_EMAIL = os.getenv("ADMIN_EMAIL", "admin@agentk.ai")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "agentk2025")

# Kubernetes
KUBECONFIG_PATH = os.getenv("KUBECONFIG", None)

# Premium features
PREMIUM_ENABLED = os.getenv("PREMIUM_ENABLED", "false").lower() == "true"
