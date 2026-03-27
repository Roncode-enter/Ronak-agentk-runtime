import { useEffect, useState } from 'react'
import { api } from '../api'

export default function Attestation() {
  const [attestations, setAttestations] = useState<any[]>([])
  const [selected, setSelected] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.listAttestations().then(r => setAttestations(r.attestations || [])).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const loadDetail = async (name: string) => {
    try {
      const data = await api.getAttestation(name)
      setSelected(data)
    } catch {}
  }

  if (loading) return <div className="text-gray-400 text-center py-20">Loading attestation data...</div>

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">TEE Attestation Reports</h2>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="space-y-4">
          {attestations.length === 0 ? (
            <div className="bg-white rounded-xl border border-gray-200 p-8 text-center text-gray-400">
              No ConfidentialAgents deployed. Create one with TEE hardware attestation.
            </div>
          ) : attestations.map(a => (
            <div
              key={a.name}
              onClick={() => loadDetail(a.name)}
              className={`bg-white rounded-xl border p-6 cursor-pointer transition hover:shadow-md ${
                selected?.name === a.name ? 'border-agentk-500 ring-2 ring-agentk-100' : 'border-gray-200'
              }`}
            >
              <div className="flex justify-between items-center">
                <div>
                  <h3 className="font-semibold text-gray-900">{a.name}</h3>
                  <p className="text-sm text-gray-500 mt-1">Agent: {a.agentRef} | TEE: {a.teeProvider}</p>
                </div>
                <span className={`px-3 py-1 rounded-full text-xs font-medium ${
                  a.verified ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'
                }`}>
                  {a.verified ? 'VERIFIED' : 'UNVERIFIED'}
                </span>
              </div>
              {a.lastAttestationTime && (
                <p className="text-xs text-gray-400 mt-2">Last attestation: {a.lastAttestationTime}</p>
              )}
            </div>
          ))}
        </div>

        {selected && (
          <div className="bg-white rounded-xl border border-gray-200 p-6">
            <h3 className="text-lg font-semibold text-gray-900 mb-4">Attestation Detail: {selected.name}</h3>
            <dl className="space-y-3">
              <DetailRow label="TEE Provider" value={selected.teeProvider} />
              <DetailRow label="Verified" value={selected.verified ? 'YES (ECDSA signature verified)' : 'NO'} />
              <DetailRow label="Referenced Agent" value={selected.agentRef} />
              <DetailRow label="Runtime Class" value={selected.runtimeClassName} />
              <DetailRow label="Memory Encryption" value={selected.memoryEncryption ? 'Enabled' : 'Disabled'} />
              <DetailRow label="Enclave Memory" value={`${selected.enclaveMemoryMB} MB`} />
              <DetailRow label="Deployment" value={selected.deploymentName} />
              <DetailRow label="Last Attestation" value={selected.lastAttestationTime} />
            </dl>
            {selected.attestationReport && (
              <div className="mt-4">
                <p className="text-sm font-medium text-gray-700 mb-2">Attestation Digest</p>
                <pre className="bg-gray-50 rounded-lg p-4 text-xs font-mono text-gray-600 break-all overflow-x-auto">
                  {selected.attestationReport}
                </pre>
              </div>
            )}
            <button
              onClick={() => {
                const blob = new Blob([JSON.stringify(selected, null, 2)], { type: 'application/json' })
                const url = URL.createObjectURL(blob)
                const a = document.createElement('a')
                a.href = url; a.download = `attestation-${selected.name}.json`; a.click()
              }}
              className="mt-4 px-4 py-2 bg-agentk-600 text-white text-sm rounded-lg hover:bg-agentk-700 transition"
            >
              Download Report
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

function DetailRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between py-2 border-b border-gray-100">
      <dt className="text-sm text-gray-500">{label}</dt>
      <dd className="text-sm font-medium text-gray-900">{value || 'n/a'}</dd>
    </div>
  )
}
