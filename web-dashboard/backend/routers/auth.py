"""Authentication router — login + kubeconfig upload."""
import io
import tempfile
from datetime import datetime, timedelta, timezone
from fastapi import APIRouter, HTTPException, UploadFile, File, Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from jose import jwt, JWTError
from pydantic import BaseModel

from backend.config import JWT_SECRET, JWT_ALGORITHM, JWT_EXPIRY_HOURS, ADMIN_EMAIL, ADMIN_PASSWORD
from backend.services.k8s_client import set_kubeconfig_data

router = APIRouter()
security = HTTPBearer()


class LoginRequest(BaseModel):
    email: str
    password: str


class LoginResponse(BaseModel):
    token: str
    email: str
    premium: bool


def create_token(email: str, premium: bool = False) -> str:
    payload = {
        "sub": email,
        "premium": premium,
        "exp": datetime.now(timezone.utc) + timedelta(hours=JWT_EXPIRY_HOURS),
    }
    return jwt.encode(payload, JWT_SECRET, algorithm=JWT_ALGORITHM)


def get_current_user(credentials: HTTPAuthorizationCredentials = Depends(security)) -> dict:
    try:
        payload = jwt.decode(credentials.credentials, JWT_SECRET, algorithms=[JWT_ALGORITHM])
        return payload
    except JWTError:
        raise HTTPException(status_code=401, detail="Invalid token")


@router.post("/login", response_model=LoginResponse)
def login(req: LoginRequest):
    if req.email == ADMIN_EMAIL and req.password == ADMIN_PASSWORD:
        token = create_token(req.email, premium=True)
        return LoginResponse(token=token, email=req.email, premium=True)
    raise HTTPException(status_code=401, detail="Invalid credentials")


@router.post("/kubeconfig")
async def upload_kubeconfig(
    file: UploadFile = File(...),
    user: dict = Depends(get_current_user),
):
    """Upload a kubeconfig file to connect to a Kubernetes cluster."""
    content = await file.read()
    try:
        set_kubeconfig_data(content.decode("utf-8"))
        return {"status": "connected", "message": "Kubeconfig loaded successfully"}
    except Exception as e:
        raise HTTPException(status_code=400, detail=f"Invalid kubeconfig: {str(e)}")
