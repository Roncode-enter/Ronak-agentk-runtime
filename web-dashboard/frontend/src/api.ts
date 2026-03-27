const API_BASE = '/api';

function getToken(): string | null {
  return localStorage.getItem('agentk_token');
}

async function fetchAPI(path: string, options: RequestInit = {}): Promise<any> {
  const token = getToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(options.headers as Record<string, string> || {}),
  };
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const res = await fetch(`${API_BASE}${path}`, { ...options, headers });
  if (res.status === 401) {
    localStorage.removeItem('agentk_token');
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ detail: res.statusText }));
    throw new Error(err.detail || 'Request failed');
  }
  return res.json();
}

export const api = {
  login: (email: string, password: string) =>
    fetchAPI('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),

  listAgents: (ns = 'default') => fetchAPI(`/agents?namespace=${ns}`),
  getAgent: (name: string, ns = 'default') => fetchAPI(`/agents/${name}?namespace=${ns}`),
  createAgent: (data: any) => fetchAPI('/agents', { method: 'POST', body: JSON.stringify(data) }),
  deleteAgent: (name: string, ns = 'default') => fetchAPI(`/agents/${name}?namespace=${ns}`, { method: 'DELETE' }),

  getCost: (name: string, ns = 'default') => fetchAPI(`/cost/${name}?namespace=${ns}`),
  getAllCosts: (ns = 'default') => fetchAPI(`/cost?namespace=${ns}`),

  getAttestation: (name: string, ns = 'default') => fetchAPI(`/attestation/${name}?namespace=${ns}`),
  listAttestations: (ns = 'default') => fetchAPI(`/attestation?namespace=${ns}`),

  previewSimulation: (data: any) => fetchAPI('/simulation/preview', { method: 'POST', body: JSON.stringify(data) }),

  uploadKubeconfig: async (file: File) => {
    const formData = new FormData();
    formData.append('file', file);
    const token = getToken();
    const res = await fetch(`${API_BASE}/auth/kubeconfig`, {
      method: 'POST',
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      body: formData,
    });
    if (!res.ok) throw new Error('Failed to upload kubeconfig');
    return res.json();
  },
};
