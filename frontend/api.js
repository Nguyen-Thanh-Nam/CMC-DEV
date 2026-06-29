import { API_URL } from './config.js';

/**
 * Fetch asset stats summary from backend.
 */
export async function getStats() {
    const res = await fetch(`${API_URL}/assets/stats`);
    if (!res.ok) throw new Error('Failed to fetch stats');
    return await res.json();
}

/**
 * Fetch paginated assets list with optional filters.
 */
export async function getAssets(page, limit, filters) {
    let queryParams = `page=${page}&limit=${limit}&type=${filters.type}&status=${filters.status}&tag=${filters.tag}`;
    const res = await fetch(`${API_URL}/assets?${queryParams}`);
    if (!res.ok) throw new Error('Failed to fetch assets');
    return await res.json();
}

/**
 * Search assets by query string.
 */
export async function searchAssets(query) {
    const res = await fetch(`${API_URL}/assets/search?q=${encodeURIComponent(query)}`);
    if (!res.ok) throw new Error('Search failed');
    return await res.json();
}

/**
 * Add a new asset to the backend.
 */
export async function createAsset(assetInput) {
    const res = await fetch(`${API_URL}/assets`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(assetInput)
    });
    if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Failed to add asset');
    }
    return await res.json();
}

/**
 * Delete a specific asset by ID.
 */
export async function deleteAsset(id) {
    const res = await fetch(`${API_URL}/assets/batch?ids=${id}`, {
        method: 'DELETE'
    });
    if (!res.ok) throw new Error('Failed to delete asset');
    return true;
}

/**
 * Trigger an attack surface scan for an asset.
 */
export async function triggerScan(assetId, scanType) {
    const res = await fetch(`${API_URL}/assets/${assetId}/scan`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ scan_type: scanType })
    });
    if (!res.ok) {
        const err = await res.json();
        throw new Error(err.error || 'Failed to trigger scan');
    }
    return await res.json();
}

/**
 * Get status and metadata of a scan job.
 */
export async function getScanJob(jobId) {
    const res = await fetch(`${API_URL}/scan-jobs/${jobId}`);
    if (!res.ok) throw new Error('Failed to fetch scan job details');
    return await res.json();
}

/**
 * Get detailed scan results of a completed scan job.
 */
export async function getScanResults(jobId) {
    const res = await fetch(`${API_URL}/scan-jobs/${jobId}/results`);
    if (!res.ok) throw new Error('Failed to fetch scan results');
    return await res.json();
}

/**
 * Get scan history log for a specific asset.
 */
export async function getAssetScans(assetId) {
    const res = await fetch(`${API_URL}/assets/${assetId}/scans`);
    if (!res.ok) throw new Error('Failed to fetch scans for asset');
    return await res.json();
}

/**
 * Get global list of all scan jobs.
 */
export async function getGlobalScanJobs() {
    const res = await fetch(`${API_URL}/scan-jobs`);
    if (!res.ok) throw new Error('Failed to fetch global scan logs');
    return await res.json();
}
