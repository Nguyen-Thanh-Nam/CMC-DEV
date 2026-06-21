import { API_URL } from './config.js';
import * as api from './api.js';
import {
    dom,
    showModal,
    hideModal,
    showToast,
    debounce,
    populateScanOptions,
    renderAssetsList,
    renderAssetScansHistory,
    renderScanResults,
    renderGlobalScanJobs
} from './ui.js';

// State Management
let currentPage = 1;
const limit = 10;
let totalAssets = 0;
let currentFilters = {
    type: '',
    status: '',
    tag: '',
    q: ''
};
let pollingIntervals = {};
let activeScanAssetId = null;

// Init
document.addEventListener('DOMContentLoaded', () => {
    fetchStats();
    fetchAssets();
    setupEventListeners();
});

// Event Listeners setup
function setupEventListeners() {
    // Filters & Search
    dom.searchInput.addEventListener('input', debounce(() => {
        currentFilters.q = dom.searchInput.value;
        currentPage = 1;
        fetchAssets();
    }, 400));

    dom.filterType.addEventListener('change', () => {
        currentFilters.type = dom.filterType.value;
        currentPage = 1;
        fetchAssets();
    });

    dom.filterStatus.addEventListener('change', () => {
        currentFilters.status = dom.filterStatus.value;
        currentPage = 1;
        fetchAssets();
    });

    dom.filterTag.addEventListener('input', debounce(() => {
        currentFilters.tag = dom.filterTag.value;
        currentPage = 1;
        fetchAssets();
    }, 400));

    dom.btnResetFilters.addEventListener('click', () => {
        dom.searchInput.value = '';
        dom.filterType.value = '';
        dom.filterStatus.value = '';
        dom.filterTag.value = '';
        currentFilters = { type: '', status: '', tag: '', q: '' };
        currentPage = 1;
        fetchAssets();
    });

    // Pagination
    dom.btnPrevPage.addEventListener('click', () => {
        if (currentPage > 1) {
            currentPage--;
            fetchAssets();
        }
    });

    dom.btnNextPage.addEventListener('click', () => {
        const totalPages = Math.ceil(totalAssets / limit);
        if (currentPage < totalPages) {
            currentPage++;
            fetchAssets();
        }
    });

    // Export CSV
    dom.btnExport.addEventListener('click', () => {
        let queryParams = `type=${currentFilters.type}&status=${currentFilters.status}&tag=${currentFilters.tag}`;
        window.open(`${API_URL}/assets/csv/export?${queryParams}`);
        showToast('CSV export started', 'success');
    });

    // Modals control
    dom.btnAddAssetModal.addEventListener('click', () => showModal(dom.addAssetModal));
    dom.btnCloseAddModal.addEventListener('click', () => hideModal(dom.addAssetModal));
    dom.btnCancelAddModal.addEventListener('click', () => hideModal(dom.addAssetModal));
    
    dom.addAssetForm.addEventListener('submit', handleAddAsset);

    // Sidebar Navigation
    const navLinks = document.querySelectorAll('.nav-links a');
    navLinks.forEach(link => {
        link.addEventListener('click', (e) => {
            e.preventDefault();
            navLinks.forEach(l => l.classList.remove('active'));
            link.classList.add('active');

            const target = link.getAttribute('href');
            if (target === '#scans-section') {
                dom.statsGrid.style.display = 'none';
                dom.filtersCard.style.display = 'none';
                dom.assetsSection.style.display = 'none';
                dom.scansSection.style.display = 'block';
                fetchGlobalScanJobs();
            } else {
                dom.statsGrid.style.display = 'grid';
                dom.filtersCard.style.display = 'block';
                dom.assetsSection.style.display = 'block';
                dom.scansSection.style.display = 'none';
            }
        });
    });

    if (dom.btnRefreshScans) {
        dom.btnRefreshScans.addEventListener('click', () => {
            fetchGlobalScanJobs();
        });
    }

    // Trigger Scan Modal close
    dom.btnCloseScanModal.addEventListener('click', () => hideModal(dom.triggerScanModal));
    dom.btnCancelScanModal.addEventListener('click', () => hideModal(dom.triggerScanModal));

    // Run Scan Handler
    dom.btnRunScan.addEventListener('click', async () => {
        if (!activeScanAssetId) return;
        const scanType = dom.scanTypeSelect.value;

        try {
            const job = await api.triggerScan(activeScanAssetId, scanType);
            showToast('Scan queued and started in background!', 'success');
            hideModal(dom.triggerScanModal);

            // Start polling for this job status
            pollJobStatus(job.id);
        } catch (err) {
            showToast(err.message, 'error');
        }
    });

    // Results Modal close
    dom.btnCloseResultsModal.addEventListener('click', () => hideModal(dom.scanResultsModal));
    dom.btnCloseResultsFooter.addEventListener('click', () => hideModal(dom.scanResultsModal));
}

// Fetch Stats
async function fetchStats() {
    try {
        const data = await api.getStats();
        dom.statTotal.textContent = data.total || 0;
        dom.statDomains.textContent = data.by_type.domain || 0;
        dom.statIps.textContent = data.by_type.ip || 0;
    } catch (e) {
        console.error('Error fetching stats:', e);
    }
}

// Fetch Assets list
async function fetchAssets() {
    dom.assetsTableBody.innerHTML = `
        <tr>
            <td colspan="6" class="text-center py-5">
                <i class="fa-solid fa-spinner fa-spin loader-icon"></i> Loading assets...
            </td>
        </tr>
    `;

    try {
        let assets, total, page;
        if (currentFilters.q) {
            const data = await api.searchAssets(currentFilters.q);
            assets = data;
            total = data.length;
            page = 1;
        } else {
            const data = await api.getAssets(currentPage, limit, currentFilters);
            assets = data.data;
            total = data.pagination.total;
            page = data.pagination.page;
        }

        renderAssetsList(assets, total, page, limit, {
            onOpenScan: (id, name, type) => {
                activeScanAssetId = id;
                populateScanOptions(name, type, id);
            },
            onViewScans: (id) => {
                viewAssetScans(id);
            },
            onDelete: (id) => {
                deleteAsset(id);
            }
        });
    } catch (e) {
        dom.assetsTableBody.innerHTML = `
            <tr>
                <td colspan="6" class="text-center py-5 text-red-500">
                    <i class="fa-solid fa-triangle-exclamation"></i> Error loading assets. Is the server running?
                </td>
            </tr>
        `;
        showToast('Failed to connect to backend API', 'error');
    }
}

// Add Asset Handler
async function handleAddAsset(e) {
    e.preventDefault();
    const input = {
        name: document.getElementById('asset-name').value.trim(),
        type: document.getElementById('asset-type').value,
        status: document.getElementById('asset-status').value,
        tags: document.getElementById('asset-tags').value.trim()
    };

    try {
        await api.createAsset(input);
        showToast('Asset added successfully!', 'success');
        hideModal(dom.addAssetModal);
        dom.addAssetForm.reset();
        fetchStats();
        fetchAssets();
    } catch (err) {
        showToast(err.message, 'error');
    }
}

// Delete Asset
async function deleteAsset(id) {
    if (!confirm('Are you sure you want to delete this asset?')) return;

    try {
        await api.deleteAsset(id);
        showToast('Asset deleted successfully', 'success');
        fetchStats();
        fetchAssets();
    } catch (e) {
        showToast(e.message, 'error');
    }
}

// Poll Scan Job Status
function pollJobStatus(jobId) {
    // Show active scan stat increments
    dom.statActive.textContent = parseInt(dom.statActive.textContent) + 1;

    pollingIntervals[jobId] = setInterval(async () => {
        try {
            const job = await api.getScanJob(jobId);

            if (job.status === 'completed' || job.status === 'failed') {
                clearInterval(pollingIntervals[jobId]);
                delete pollingIntervals[jobId];
                
                // Decrement active scans stat
                dom.statActive.textContent = Math.max(0, parseInt(dom.statActive.textContent) - 1);
                
                showToast(`Scan job ${job.status === 'completed' ? 'completed successfully' : 'failed'}!`, job.status === 'completed' ? 'success' : 'error');
                
                if (job.status === 'completed') {
                    // Automatically pop results modal
                    viewScanResults(jobId);
                }
            }
        } catch (e) {
            console.error(e);
        }
    }, 2000);
}

// View Asset Scans history
async function viewAssetScans(assetId) {
    dom.resultsContentArea.innerHTML = `<div class="text-center py-5"><i class="fa-solid fa-spinner fa-spin loader-icon"></i> Fetching scan history...</div>`;
    dom.resultsJobId.textContent = 'Asset Log';
    dom.resultsStatus.textContent = '-';
    dom.resultsTime.textContent = '-';
    showModal(dom.scanResultsModal);

    try {
        const jobs = await api.getAssetScans(assetId);
        renderAssetScansHistory(jobs, {
            onViewResults: (jobId) => {
                viewScanResults(jobId);
            }
        });
    } catch (e) {
        dom.resultsContentArea.innerHTML = `<p class="text-red-500 text-center py-5">Failed to load scan jobs</p>`;
    }
}

// View Scan Results for a specific Job
async function viewScanResults(jobId) {
    dom.resultsContentArea.innerHTML = `<div class="text-center py-5"><i class="fa-solid fa-spinner fa-spin loader-icon"></i> Fetching scan data...</div>`;
    showModal(dom.scanResultsModal);

    try {
        const job = await api.getScanJob(jobId);
        const data = await api.getScanResults(jobId);
        renderScanResults(job, data);
    } catch (e) {
        dom.resultsContentArea.innerHTML = `<p class="text-red-500 text-center py-5">Failed to load scan job details: ${e.message}</p>`;
    }
}

// Fetch all scan jobs globally
async function fetchGlobalScanJobs() {
    dom.scansTableBody.innerHTML = `<tr><td colspan="6" class="text-center py-5"><i class="fa-solid fa-spinner fa-spin loader-icon"></i> Loading global scan logs...</td></tr>`;
    try {
        const jobs = await api.getGlobalScanJobs();
        renderGlobalScanJobs(jobs, {
            onViewResults: (jobId) => {
                viewScanResults(jobId);
            }
        });
    } catch (err) {
        dom.scansTableBody.innerHTML = `<tr><td colspan="6" class="text-center py-5 text-red-500"><i class="fa-solid fa-triangle-exclamation"></i> Failed to fetch scan logs. Make sure API is running.</td></tr>`;
    }
}
