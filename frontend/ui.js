// DOM Elements
export const dom = {
    assetsTableBody: document.getElementById('assets-table-body'),
    searchInput: document.getElementById('search-input'),
    filterType: document.getElementById('filter-type'),
    filterStatus: document.getElementById('filter-status'),
    filterTag: document.getElementById('filter-tag'),
    btnResetFilters: document.getElementById('btn-reset-filters'),
    btnExport: document.getElementById('btn-export'),

    statTotal: document.getElementById('stat-total'),
    statsGrid: document.querySelector('.stats-grid'),
    filtersCard: document.querySelector('.filters-card'),
    assetsSection: document.getElementById('assets-section'),
    scansSection: document.getElementById('scans-section'),
    scansTableBody: document.getElementById('scans-table-body'),
    btnRefreshScans: document.getElementById('btn-refresh-scans'),
    statDomains: document.getElementById('stat-domains'),
    statIps: document.getElementById('stat-ips'),
    statActive: document.getElementById('stat-active'),

    paginationInfo: document.getElementById('pagination-info'),
    btnPrevPage: document.getElementById('btn-prev-page'),
    btnNextPage: document.getElementById('btn-next-page'),

    // Modals
    addAssetModal: document.getElementById('add-asset-modal'),
    btnAddAssetModal: document.getElementById('btn-add-asset-modal'),
    btnCloseAddModal: document.getElementById('btn-close-add-modal'),
    btnCancelAddModal: document.getElementById('btn-cancel-add-modal'),
    addAssetForm: document.getElementById('add-asset-form'),

    triggerScanModal: document.getElementById('trigger-scan-modal'),
    btnCloseScanModal: document.getElementById('btn-close-scan-modal'),
    btnCancelScanModal: document.getElementById('btn-cancel-scan-modal'),
    btnRunScan: document.getElementById('btn-run-scan'),
    scanTypeSelect: document.getElementById('scan-type-select'),
    scanAssetName: document.getElementById('scan-asset-name'),
    scanAssetType: document.getElementById('scan-asset-type'),

    scanResultsModal: document.getElementById('scan-results-modal'),
    btnCloseResultsModal: document.getElementById('btn-close-results-modal'),
    btnCloseResultsFooter: document.getElementById('btn-close-results-footer'),
    resultsContentArea: document.getElementById('results-content-area'),
    resultsJobId: document.getElementById('results-job-id'),
    resultsStatus: document.getElementById('results-status'),
    resultsTime: document.getElementById('results-time'),

    // Toast
    toast: document.getElementById('toast-notification')
};

// Global callbacks map for inline onclick attributes
window.uiCallbacks = {};

/**
 * Show a modal element.
 */
export function showModal(modal) {
    modal.classList.add('show');
}

/**
 * Hide a modal element.
 */
export function hideModal(modal) {
    modal.classList.remove('show');
}

/**
 * Display a toast notification.
 */
export function showToast(message, type = 'success') {
    dom.toast.textContent = message;
    dom.toast.className = `toast show ${type}`;

    setTimeout(() => {
        dom.toast.classList.remove('show');
    }, 3500);
}

/**
 * Debounce helper function.
 */
export function debounce(func, wait) {
    let timeout;
    return function() {
        const context = this, args = arguments;
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(context, args), wait);
    };
}

/**
 * Populate the scan options dropdown based on asset type and save asset ID.
 */
export function populateScanOptions(name, type, assetId, onRunScan) {
    dom.scanAssetName.textContent = name;
    dom.scanAssetType.textContent = type;
    dom.scanAssetType.className = `badge ${type}`;

    // Populate dropdown options
    if (type === 'domain') {
        dom.scanTypeSelect.innerHTML = `
            <option value="all">All Passive Scans (DNS, WHOIS, SSL, Tech)</option>
            <option value="dns">DNS Records Lookup</option>
            <option value="whois">WHOIS Registration</option>
            <option value="asn">ASN Lookup</option>
            <option value="ssl">SSL/TLS Cert Scan</option>
            <option value="tech">Technology Detection</option>
            <option value="port">Port Scan (Safety Filtered)</option>
        `;
    } else if (type === 'ip') {
        dom.scanTypeSelect.innerHTML = `
            <option value="ip">IP Geolocation & ASN</option>
            <option value="asn">ASN Lookup</option>
            <option value="port">Port Scan (Safety Filtered)</option>
        `;
    } else {
        dom.scanTypeSelect.innerHTML = `
            <option value="port">Port Scan (Safety Filtered)</option>
        `;
    }

    showModal(dom.triggerScanModal);
}

/**
 * Render list of assets in the main inventory table.
 */
export function renderAssetsList(assets, total, page, limit, callbacks) {
    // Save callbacks to global window object so inline HTML 'onclick' can reference them
    window.uiCallbacks.openScanModal = callbacks.onOpenScan;
    window.uiCallbacks.viewAssetScans = callbacks.onViewScans;
    window.uiCallbacks.deleteAsset = callbacks.onDelete;

    if (!assets || assets.length === 0) {
        dom.assetsTableBody.innerHTML = `
            <tr>
                <td colspan="6" class="text-center py-5 text-gray-500">
                    No assets found. Click "New Asset" to add one!
                </td>
            </tr>
        `;
        dom.paginationInfo.textContent = 'Showing 0-0 of 0';
        dom.btnPrevPage.disabled = true;
        dom.btnNextPage.disabled = true;
        return;
    }

    // Render rows
    dom.assetsTableBody.innerHTML = assets.map(asset => {
        const date = new Date(asset.created_at).toLocaleString();
        const typeBadge = `<span class="badge ${asset.type}">${asset.type}</span>`;
        const statusBadge = `<span class="badge ${asset.status}">${asset.status}</span>`;
        
        const tagsHTML = asset.tags 
            ? asset.tags.split(',').map(t => `<span class="tag-badge">${t.trim()}</span>`).join('')
            : '<span class="text-gray-500">-</span>';

        return `
            <tr id="asset-row-${asset.id}">
                <td><strong>${asset.name}</strong></td>
                <td>${typeBadge}</td>
                <td>${statusBadge}</td>
                <td>${tagsHTML}</td>
                <td>${date}</td>
                <td>
                    <div class="header-actions">
                        <button class="btn btn-icon" onclick="window.uiCallbacks.openScanModal('${asset.id}', '${asset.name}', '${asset.type}')" title="Scan Asset">
                            <i class="fa-solid fa-satellite-dish text-indigo-400"></i>
                        </button>
                        <button class="btn btn-icon" onclick="window.uiCallbacks.viewAssetScans('${asset.id}')" title="Scan Results">
                            <i class="fa-solid fa-chart-line"></i>
                        </button>
                        <button class="btn btn-icon" onclick="window.uiCallbacks.deleteAsset('${asset.id}')" title="Delete Asset">
                            <i class="fa-solid fa-trash text-red-400"></i>
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');

    // Update Pagination Controls
    const totalPages = Math.ceil(total / limit);
    const start = (page - 1) * limit + 1;
    const end = Math.min(page * limit, total);
    dom.paginationInfo.textContent = `Showing ${start}-${end} of ${total}`;
    
    dom.btnPrevPage.disabled = page <= 1;
    dom.btnNextPage.disabled = page >= totalPages;
}

/**
 * Render history log for a specific asset.
 */
export function renderAssetScansHistory(jobs, callbacks) {
    window.uiCallbacks.viewScanResults = callbacks.onViewResults;

    if (!jobs || jobs.length === 0) {
        dom.resultsContentArea.innerHTML = `<p class="text-gray-400 text-center py-5">No scan jobs found for this asset. Start a new scan to get data!</p>`;
        return;
    }

    dom.resultsContentArea.innerHTML = `
        <h4>Scan History Log</h4>
        <div class="table-responsive">
            <table class="assets-table">
                <thead>
                    <tr>
                        <th>Scan Type</th>
                        <th>Status</th>
                        <th>Results Count</th>
                        <th>Started At</th>
                        <th>Action</th>
                    </tr>
                </thead>
                <tbody>
                    ${jobs.map(job => {
                        const date = job.started_at ? new Date(job.started_at).toLocaleString() : 'N/A';
                        const statusColor = job.status === 'completed' ? 'success' : (job.status === 'failed' ? 'error' : 'warning');
                        return `
                            <tr>
                                <td><strong>${job.scan_type.toUpperCase()}</strong></td>
                                <td><span class="badge ${statusColor}">${job.status}</span></td>
                                <td>${job.results_count}</td>
                                <td>${date}</td>
                                <td>
                                    ${job.status === 'completed' 
                                        ? `<button class="btn btn-primary" style="padding: 0.4rem 0.8rem; font-size: 0.8rem;" onclick="window.uiCallbacks.viewScanResults('${job.id}')">View Data</button>`
                                        : `<span class="text-gray-500">-</span>`}
                                </td>
                            </tr>
                        `;
                    }).join('')}
                </tbody>
            </table>
        </div>
    `;
}

/**
 * Render detailed scan results inside the results modal.
 */
export function renderScanResults(job, data) {
    dom.resultsJobId.textContent = job.id;
    dom.resultsStatus.textContent = job.status.toUpperCase();
    dom.resultsStatus.className = `badge ${job.status === 'completed' ? 'active' : 'inactive'}`;
    dom.resultsTime.textContent = job.ended_at ? new Date(job.ended_at).toLocaleString() : 'N/A';

    if (!data.results || data.results.length === 0) {
        dom.resultsContentArea.innerHTML = `<p class="text-gray-400 text-center">No details available for this job.</p>`;
        return;
    }

    let html = '';
    data.results.forEach(resRecord => {
        const res = resRecord.result_data;
        
        // Render depending on scan type/fields
        if (job.scan_type === 'dns' || res.a || res.mx || res.ns || res.txt) {
            html += `
                <h4>DNS Lookup Details</h4>
                <div class="results-grid">
                    <div class="results-card">
                        <h5>A Records (IPs)</h5>
                        <div class="results-list">${res.a ? res.a.map(x => `<div class="results-list-item">${x}</div>`).join('') : '<span class="text-gray-500">None</span>'}</div>
                    </div>
                    <div class="results-card">
                        <h5>MX Records (Mail)</h5>
                        <div class="results-list">${res.mx ? res.mx.map(x => `<div class="results-list-item">${x}</div>`).join('') : '<span class="text-gray-500">None</span>'}</div>
                    </div>
                    <div class="results-card">
                        <h5>NS Records (Name Server)</h5>
                        <div class="results-list">${res.ns ? res.ns.map(x => `<div class="results-list-item">${x}</div>`).join('') : '<span class="text-gray-500">None</span>'}</div>
                    </div>
                </div>
            `;
        } 
        else if (job.scan_type === 'whois' || res.registrar) {
            html += `
                <h4>WHOIS Query Result</h4>
                <div class="results-grid" style="margin-bottom: 1.5rem;">
                    <div class="results-card">
                        <h5>Registrar</h5>
                        <p>${res.registrar || 'N/A'}</p>
                    </div>
                    <div class="results-card">
                        <h5>Registry Domain ID</h5>
                        <p>${res.registry_domain_id || 'N/A'}</p>
                    </div>
                </div>
                ${res.raw ? `
                    <h5>Raw WHOIS Output</h5>
                    <pre style="background: #05060a; padding: 1rem; border-radius: 6px; overflow-x: auto;">${res.raw}</pre>
                ` : ''}
            `;
        }
        else if (job.scan_type === 'ip' || job.scan_type === 'asn' || res.geolocation) {
            html += `
                <h4>IP ASN & Geolocation Details</h4>
                <div class="results-grid" style="margin-bottom: 1.5rem;">
                    <div class="results-card">
                        <h5>IP Address</h5>
                        <p>${res.ip_address || 'N/A'}</p>
                    </div>
                    <div class="results-card">
                        <h5>Reverse DNS</h5>
                        <p>${res.reverse_dns || 'N/A'}</p>
                    </div>
                    <div class="results-card">
                        <h5>Country</h5>
                        <p>${res.geolocation?.country || 'N/A'} (${res.geolocation?.country_code || 'N/A'})</p>
                    </div>
                    <div class="results-card">
                        <h5>City / Region</h5>
                        <p>${res.geolocation?.city || 'N/A'}, ${res.geolocation?.region || 'N/A'}</p>
                    </div>
                    <div class="results-card">
                        <h5>ISP / Org</h5>
                        <p>${res.geolocation?.isp || 'N/A'}</p>
                    </div>
                    <div class="results-card">
                        <h5>ASN</h5>
                        <p>AS${res.asn?.number || '0'} (${res.asn?.name || 'N/A'})</p>
                    </div>
                </div>
            `;
        }
        else if (job.scan_type === 'port' || res.open_ports) {
            html += `
                <h4>Port Scan Results (Safety Filtered)</h4>
                <p style="margin-bottom: 1rem; font-size: 0.85rem; color: var(--text-muted);">
                    Scanned target: <strong>${res.ip_address}</strong>. 
                    Scanned ports count: ${res.total_scanned}. 
                    Closed ports: ${res.closed_ports}. 
                    Duration: ${res.scan_duration_ms} ms.
                </p>
                ${res.open_ports && res.open_ports.length > 0 ? `
                    <div class="table-responsive">
                        <table class="assets-table">
                            <thead>
                                <tr>
                                    <th>Port</th>
                                    <th>Protocol</th>
                                    <th>State</th>
                                    <th>Detected Service</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${res.open_ports.map(p => `
                                    <tr>
                                        <td><strong>${p.port}</strong></td>
                                        <td>${p.protocol.toUpperCase()}</td>
                                        <td><span class="badge active">${p.state}</span></td>
                                        <td><span class="tag-badge">${p.service}</span></td>
                                    </tr>
                                `).join('')}
                            </tbody>
                        </table>
                    </div>
                ` : '<p class="text-center py-5 text-gray-400">All scanned ports are closed. Target appears secured.</p>'}
            `;
        }
        else if (job.scan_type === 'ssl' || res.certificate) {
            const cert = res.certificate;
            const isExp = cert.is_expired;
            html += `
                <h4>SSL/TLS Certificate Details</h4>
                <div style="display: flex; gap: 1rem; align-items: center; margin-bottom: 1.5rem;">
                    <span style="font-size: 1.75rem; font-weight: 800; color: ${isExp ? 'var(--error)' : 'var(--success)'}">Grade: ${res.grade}</span>
                    <div>
                        <span class="badge ${isExp ? 'inactive' : 'active'}">${isExp ? 'Expired' : 'Valid'}</span>
                        <span class="badge" style="background: rgba(255,255,255,0.05); margin-left: 0.25rem;">${res.connection?.tls_version}</span>
                    </div>
                </div>
                <div class="results-grid" style="margin-bottom: 1.5rem;">
                    <div class="results-card">
                        <h5>Subject CN</h5>
                        <p>${cert.subject || 'N/A'}</p>
                    </div>
                    <div class="results-card">
                        <h5>Issuer</h5>
                        <p>${cert.issuer || 'N/A'}</p>
                    </div>
                    <div class="results-card">
                        <h5>Serial Number</h5>
                        <p>${cert.serial_number || 'N/A'}</p>
                    </div>
                    <div class="results-card">
                        <h5>Expires In</h5>
                        <p>${cert.days_until_expiry} days</p>
                    </div>
                </div>
            `;
        }
        else if (job.scan_type === 'tech' || res.technologies) {
            html += `
                <h4>Technology & Stack Detection</h4>
                <div class="results-grid" style="margin-bottom: 1.5rem;">
                    ${res.technologies.map(t => `
                        <div class="results-card">
                            <h5>${t.category}</h5>
                            <p style="color: var(--primary);">${t.name} ${t.version ? `<span style="font-size: 0.8rem; color: var(--text-muted)">v${t.version}</span>` : ''}</p>
                            <small>Confidence: ${t.confidence}%</small>
                        </div>
                    `).join('')}
                </div>
                <h5>HTTP Response Headers</h5>
                <pre style="background: #05060a; padding: 1rem; border-radius: 6px; overflow-x: auto; max-height: 200px; font-size: 0.8rem;">${JSON.stringify(res.headers, null, 2)}</pre>
            `;
        }
        else {
            // Fallback raw formatting
            html += `
                <h4>Raw Data Output</h4>
                <pre>${JSON.stringify(res, null, 2)}</pre>
            `;
        }
        html += '<hr style="border: 0; border-top: 1px solid rgba(255,255,255,0.05); margin: 2rem 0;">';
    });

    dom.resultsContentArea.innerHTML = html;
}

/**
 * Render global list of all scan jobs.
 */
export function renderGlobalScanJobs(jobs, callbacks) {
    window.uiCallbacks.viewScanResults = callbacks.onViewResults;

    if (!jobs || jobs.length === 0) {
        dom.scansTableBody.innerHTML = `<tr><td colspan="6" class="text-center py-5 text-gray-400">No scan jobs found in the system.</td></tr>`;
        return;
    }

    dom.scansTableBody.innerHTML = jobs.map(job => {
        const date = job.started_at ? new Date(job.started_at).toLocaleString() : 'N/A';
        const statusColor = job.status === 'completed' ? 'success' : (job.status === 'failed' ? 'error' : 'warning');
        return `
            <tr>
                <td><code>${job.id}</code></td>
                <td><code>${job.asset_id}</code></td>
                <td><span class="tag-badge">${job.scan_type.toUpperCase()}</span></td>
                <td><span class="badge ${statusColor}">${job.status}</span></td>
                <td>${date}</td>
                <td>
                    ${job.status === 'completed' 
                        ? `<button class="btn btn-primary" style="padding: 0.4rem 0.8rem; font-size: 0.8rem;" onclick="window.uiCallbacks.viewScanResults('${job.id}')">View Results</button>`
                        : `<span class="text-gray-500">-</span>`}
                </td>
            </tr>
        `;
    }).join('');
}
