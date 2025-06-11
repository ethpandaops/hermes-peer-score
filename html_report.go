package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// OptimizedHTMLTemplateData represents the minimal data structure for the optimized HTML report.
type OptimizedHTMLTemplateData struct {
	GeneratedAt      time.Time        `json:"generated_at"`
	Summary          SummaryData      `json:"summary"`
	DataFile         string           `json:"data_file"`
	AIAnalysis       string           `json:"ai_analysis"`
	AIAnalysisHTML   template.HTML    `json:"-"`
	ValidationMode   ValidationMode   `json:"validation_mode"`
	ValidationConfig ValidationConfig `json:"validation_config"`
}

// SummaryData contains high-level summary information for the report.
type SummaryData struct {
	TestDuration         float64       `json:"test_duration"`
	StartTime            time.Time     `json:"start_time"`
	EndTime              time.Time     `json:"end_time"`
	TotalConnections     int           `json:"total_connections"`
	SuccessfulHandshakes int           `json:"successful_handshakes"`
	FailedHandshakes     int           `json:"failed_handshakes"`
	UniquePeers          int           `json:"unique_peers"`
	PeerSummaries        []PeerSummary `json:"peer_summaries"`
}

// PeerSummary contains minimal information about a peer for the overview.
type PeerSummary struct {
	PeerID            string  `json:"peer_id"`
	ShortPeerID       string  `json:"short_peer_id"`
	ClientType        string  `json:"client_type"`
	ClientAgent       string  `json:"client_agent"`
	SessionCount      int     `json:"session_count"`
	EventCount        int     `json:"event_count"`
	GoodbyeCount      int     `json:"goodbye_count"`
	MeshCount         int     `json:"mesh_count"`
	MinPeerScore      float64 `json:"min_peer_score"`
	MaxPeerScore      float64 `json:"max_peer_score"`
	HasScores         bool    `json:"has_scores"`
	LastSessionStatus string  `json:"last_session_status"`
	LastSessionTime   string  `json:"last_session_time"`
}

// optimizedHTMLTemplate contains the optimized HTML template that loads data dynamically.
const optimizedHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hermes Peer Score Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        .loading { opacity: 0.5; pointer-events: none; }
        .peer-card { cursor: pointer; transition: all 0.2s; }
        .peer-card:hover { transform: translateY(-2px); box-shadow: 0 4px 20px rgba(0,0,0,0.1); }
        .pagination { user-select: none; }
        .score-positive { color: #10b981; }
        .score-negative { color: #ef4444; }
        .score-neutral { color: #6b7280; }
        .virtual-scroll-container { height: 600px; overflow-y: auto; }
        .peer-item { min-height: 120px; }
        .detail-panel { max-height: 80vh; overflow-y: auto; }
        .client-logo { transition: all 0.2s ease; }
        .client-logo:hover { transform: scale(1.05); }
        .client-fallback {
            background: linear-gradient(45deg, #3b82f6, #1d4ed8);
            font-weight: bold;
            text-shadow: 0 1px 2px rgba(0,0,0,0.3);
        }
        .ai-analysis-content {
            line-height: 1.6;
        }
        /* Validation mode specific styles */
        .validation-mode-delegated {
            --validation-primary: #2563eb;
            --validation-secondary: #dbeafe;
            --validation-accent: #1d4ed8;
        }
        .validation-mode-independent {
            --validation-primary: #059669;
            --validation-secondary: #d1fae5;
            --validation-accent: #047857;
        }
        .validation-badge {
            background: var(--validation-secondary);
            color: var(--validation-primary);
            border: 1px solid var(--validation-primary);
        }
        .validation-header {
            background: linear-gradient(135deg, var(--validation-primary), var(--validation-accent));
        }
        .validation-icon {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            width: 24px;
            height: 24px;
            border-radius: 50%;
            background: rgba(255, 255, 255, 0.2);
            margin-right: 8px;
        }
        .metrics-card {
            border-left: 4px solid var(--validation-primary);
            transition: all 0.2s ease;
        }
        .metrics-card:hover {
            box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
            transform: translateY(-1px);
        }
        .comparison-highlight {
            background: linear-gradient(90deg, var(--validation-secondary), transparent);
            border-left: 3px solid var(--validation-primary);
            padding-left: 12px;
        }
    </style>
</head>
<body class="bg-gray-50 min-h-screen validation-mode-{{.ValidationMode}}">
    <div class="container mx-auto px-4 py-8 max-w-7xl">
        <!-- Header with Validation Mode Branding -->
        <div class="validation-header text-white rounded-lg shadow-lg p-6 mb-6">
            <div class="flex items-center justify-between">
                <div>
                    <div class="flex items-center">
                        <div class="validation-icon">
                            {{if eq .ValidationMode "delegated"}}🔗{{else}}⚡{{end}}
                        </div>
                        <h1 class="text-3xl font-bold">Hermes Peer Score Report</h1>
                    </div>
                    <div class="flex items-center mt-2 space-x-4">
                        <span class="validation-badge px-3 py-1 rounded-full text-sm font-medium">
                            {{if eq .ValidationMode "delegated"}}Delegated Validation{{else}}Independent Validation{{end}}
                        </span>
                        <span class="text-sm opacity-90">
                            {{.ValidationConfig.HermesVersion}}
                        </span>
                        <span class="text-sm opacity-90">
                            Generated: {{.GeneratedAt.Format "January 2, 2006 at 3:04 PM"}}
                        </span>
                    </div>
                </div>
                <div class="text-right">
                    <div class="text-sm opacity-90">Test Duration</div>
                    <div class="text-2xl font-semibold">{{printf "%.1f" .Summary.TestDuration}}s</div>
                </div>
            </div>
        </div>



        <!-- Summary Statistics -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Total Connections</div>
                <div class="text-2xl font-bold text-gray-900">{{.Summary.TotalConnections}}</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Successful Handshakes</div>
                <div class="text-2xl font-bold text-green-600">{{.Summary.SuccessfulHandshakes}}</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Failed Handshakes</div>
                <div class="text-2xl font-bold text-red-600">{{.Summary.FailedHandshakes}}</div>
            </div>
            <div class="bg-white rounded-lg shadow p-6">
                <div class="text-sm font-medium text-gray-500">Unique Peers</div>
                <div class="text-2xl font-bold text-blue-600">{{.Summary.UniquePeers}}</div>
            </div>
        </div>

        <!-- Controls -->
        <div class="bg-white rounded-lg shadow p-4 mb-6">
            <div class="flex flex-wrap items-center gap-4">
                <div class="flex items-center space-x-2">
                    <label for="search" class="text-sm font-medium text-gray-700">Search:</label>
                    <input type="text" id="search" placeholder="Filter by peer ID or client..."
                           class="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
                </div>
                <div class="flex items-center space-x-2">
                    <label for="pageSize" class="text-sm font-medium text-gray-700">Show:</label>
                    <select id="pageSize" class="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
                        <option value="10">10 peers</option>
                        <option value="25" selected>25 peers</option>
                        <option value="50">50 peers</option>
                        <option value="100">100 peers</option>
                    </select>
                </div>
                <div class="flex items-center space-x-2">
                    <label for="sortBy" class="text-sm font-medium text-gray-700">Sort by:</label>
                    <select id="sortBy" class="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500">
                        <option value="events">Event Count</option>
                        <option value="sessions">Session Count</option>
                        <option value="goodbyes">Goodbye Count</option>
                        <option value="minScore">Lowest Score</option>
                        <option value="maxScore">Highest Score</option>
                        <option value="status">Session Status</option>
                        <option value="client">Client Type</option>
                    </select>
                </div>
                <div class="flex space-x-2">
                    <button onclick="exportFilteredData()" class="px-4 py-2 bg-green-600 text-white rounded-md text-sm hover:bg-green-700">
                        Export Filtered JSON
                    </button>
                    {{if .AIAnalysis}}
                    <button onclick="openAIAnalysisModal()" class="px-4 py-2 bg-blue-600 text-white rounded-md text-sm hover:bg-blue-700">
                        AI Analysis
                    </button>
                    {{end}}
                </div>
            </div>
        </div>

        <!-- Peer List -->
        <div class="bg-white rounded-lg shadow-lg">
            <div class="p-6 border-b border-gray-200">
                <h2 class="text-xl font-semibold text-gray-900">Peer Analysis</h2>
                <p class="text-gray-600 mt-1">Test ran from {{.Summary.StartTime.Format "15:04:05"}} to {{.Summary.EndTime.Format "15:04:05"}} on {{.Summary.StartTime.Format "Jan 2, 2006"}}</p>
                <div class="mt-2 text-sm text-gray-500">
                    <span id="resultsInfo">Loading...</span>
                </div>
            </div>
            <div class="p-6">
                <div id="peerList" class="space-y-4">
                    <div class="text-center py-8 text-gray-500">
                        <div class="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
                        <div id="loadingText">Loading client information and peer data...</div>
                    </div>
                </div>

                <!-- Pagination -->
                <div id="pagination" class="mt-6 flex items-center justify-between">
                    <div class="text-sm text-gray-600" id="paginationInfo"></div>
                    <div class="flex space-x-2" id="paginationControls"></div>
                </div>
            </div>
        </div>
    </div>

    <!-- Peer Detail Modal -->
    <div id="peerModal" class="fixed inset-0 bg-black bg-opacity-50 hidden z-50">
        <div class="flex items-center justify-center min-h-screen p-4">
            <div class="bg-white rounded-lg shadow-xl max-w-6xl w-full detail-panel">
                <div class="p-6 border-b border-gray-200">
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg font-semibold text-gray-900" id="modalTitle">Peer Details</h3>
                        <button onclick="closePeerModal()" class="text-gray-400 hover:text-gray-600">
                            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                            </svg>
                        </button>
                    </div>
                </div>
                <div id="modalContent" class="p-6">
                    <div class="text-center py-8 text-gray-500">
                        <div class="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>
                        Loading peer details...
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- AI Analysis Modal -->
    {{if .AIAnalysis}}
    <div id="aiAnalysisModal" class="fixed inset-0 bg-black bg-opacity-50 hidden z-50">
        <div class="flex items-center justify-center min-h-screen p-4">
            <div class="bg-white rounded-lg shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
                <div class="p-6 border-b border-gray-200">
                    <div class="flex items-center justify-between">
                        <h3 class="text-lg font-semibold text-gray-900">
                            AI Analysis
                        </h3>
                        <button onclick="closeAIAnalysisModal()" class="text-gray-400 hover:text-gray-600">
                            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                            </svg>
                        </button>
                    </div>
                </div>
                <div class="p-6 overflow-y-auto max-h-[calc(90vh-120px)]">
                    <div class="ai-analysis-content">{{.AIAnalysisHTML}}</div>
                </div>
            </div>
        </div>
    </div>
    {{end}}

    <script src="{{.DataFile}}" onerror="console.error('Failed to load data file: {{.DataFile}}')"></script>
    <script>
        let allPeers = [];
        let filteredPeers = [];
        let currentPage = 1;
        let pageSize = 25;
        let sortBy = 'events';
        let clientLogos = {};

        // Fetch client logos from ethpandaops
        async function fetchClientLogos() {
            try {
                const response = await fetch('https://ethpandaops-platform-production-cartographoor.ams3.cdn.digitaloceanspaces.com/networks.json');
                const data = await response.json();

                if (data && data.clients) {
                    for (const [clientName, clientInfo] of Object.entries(data.clients)) {
                        if (clientInfo.logo) {
                            clientLogos[clientName.toLowerCase()] = {
                                logo: clientInfo.logo,
                                displayName: clientInfo.displayName || clientName,
                                websiteUrl: clientInfo.websiteUrl
                            };
                        }
                    }
                }
            } catch (error) {
                console.warn('Failed to fetch client logos:', error);
            }
        }

        // Get client logo URL for a given client type
        function getClientLogo(clientType) {
            const normalizedClient = clientType.toLowerCase();

            // Direct match
            if (clientLogos[normalizedClient]) {
                return clientLogos[normalizedClient];
            }

            // Try partial matches for common variations
            for (const [key, value] of Object.entries(clientLogos)) {
                if (normalizedClient.includes(key) || key.includes(normalizedClient)) {
                    return value;
                }
            }

            return null;
        }

        // Initialize the application
        document.addEventListener('DOMContentLoaded', async function() {
            // Update loading message
            const loadingText = document.getElementById('loadingText');
            if (loadingText) loadingText.textContent = 'Fetching client information...';

            // Fetch client logos first
            await fetchClientLogos();
            console.log('Loaded logos for clients:', Object.keys(clientLogos));

            if (loadingText) loadingText.textContent = 'Loading peer data...';

            if (typeof reportData !== 'undefined') {
                console.log('Report data loaded successfully:', Object.keys(reportData));
                allPeers = reportData.peers || [];
                filteredPeers = [...allPeers];
                sortPeers();
                renderPeerList();
                setupEventListeners();
                updateResultsInfo();
            } else {
                console.error('reportData is undefined - data file may have failed to load');
                document.getElementById('peerList').innerHTML =
                    '<div class="text-center py-8 text-red-500">Error: Could not load peer data</div>';
            }
        });

        function setupEventListeners() {
            document.getElementById('search').addEventListener('input', debounce(handleSearch, 300));
            document.getElementById('pageSize').addEventListener('change', handlePageSizeChange);
            document.getElementById('sortBy').addEventListener('change', handleSortChange);
        }

        function debounce(func, wait) {
            let timeout;
            return function executedFunction(...args) {
                const later = () => {
                    clearTimeout(timeout);
                    func(...args);
                };
                clearTimeout(timeout);
                timeout = setTimeout(later, wait);
            };
        }

        function handleSearch(e) {
            const query = e.target.value.toLowerCase();
            filteredPeers = allPeers.filter(peer =>
                peer.peer_id.toLowerCase().includes(query) ||
                peer.client_type.toLowerCase().includes(query) ||
                peer.client_agent.toLowerCase().includes(query)
            );
            currentPage = 1;
            renderPeerList();
            updateResultsInfo();
        }

        function handlePageSizeChange(e) {
            pageSize = parseInt(e.target.value);
            currentPage = 1;
            renderPeerList();
        }

        function handleSortChange(e) {
            sortBy = e.target.value;
            sortPeers();
            renderPeerList();
        }

        function sortPeers() {
            filteredPeers.sort((a, b) => {
                switch(sortBy) {
                    case 'events': return b.event_count - a.event_count;
                    case 'sessions': return b.session_count - a.session_count;
                    case 'goodbyes': return b.goodbye_count - a.goodbye_count;
                    case 'minScore':
                        // Sort by lowest score (ascending, so worst scores first)
                        if (!a.has_scores && !b.has_scores) return 0;
                        if (!a.has_scores) return 1;
                        if (!b.has_scores) return -1;
                        return a.min_peer_score - b.min_peer_score;
                    case 'maxScore':
                        // Sort by highest score (descending, so best scores first)
                        if (!a.has_scores && !b.has_scores) return 0;
                        if (!a.has_scores) return 1;
                        if (!b.has_scores) return -1;
                        return b.max_peer_score - a.max_peer_score;
                    case 'status':
                        // Sort by session status (Connected first, then Disconnected)
                        const statusA = a.last_session_status || '';
                        const statusB = b.last_session_status || '';
                        if (statusA === statusB) return 0;
                        if (statusA === 'Connected') return -1;
                        if (statusB === 'Connected') return 1;
                        return statusA.localeCompare(statusB);
                    case 'client': return a.client_type.localeCompare(b.client_type);
                    default: return 0;
                }
            });
        }

        function renderPeerList() {
            const startIndex = (currentPage - 1) * pageSize;
            const endIndex = startIndex + pageSize;
            const pageData = filteredPeers.slice(startIndex, endIndex);

            const html = pageData.map(peer => renderPeerCard(peer)).join('');
            document.getElementById('peerList').innerHTML = html || '<div class="text-center py-8 text-gray-500">No peers found</div>';

            renderPagination();
        }

        function renderPeerCard(peer) {
            const clientInfo = getClientLogo(peer.client_type);
            const logoImg = clientInfo ?
                ` + "`" + `<img src="${clientInfo.logo}" alt="${clientInfo.displayName}" class="w-8 h-8 rounded-md object-cover client-logo" onerror="this.style.display='none'">` + "`" + ` :
                ` + "`" + `<div class="w-8 h-8 rounded-md flex items-center justify-center text-white text-xs client-fallback">${peer.client_type.substring(0, 2).toUpperCase()}</div>` + "`" + `;

            const clientDisplay = clientInfo ?
                ` + "`" + `<span class="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded" title="${clientInfo.displayName}">${peer.client_type}</span>` + "`" + ` :
                ` + "`" + `<span class="px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded">${peer.client_type}</span>` + "`" + `;

            return ` + "`" + `
                <div class="peer-card border border-gray-200 rounded-lg p-4 hover:shadow-md transition-all" onclick="showPeerDetails('${peer.peer_id}')">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center space-x-4">
                            <div class="flex-shrink-0">
                                ${logoImg}
                            </div>
                            <div class="min-w-0 flex-1">
                                <h4 class="font-medium text-gray-900">${peer.short_peer_id}...</h4>
                            </div>
                            <div class="flex flex-wrap gap-2">
                                ${clientDisplay}
                                ${peer.last_session_status ? ` + "`" + `<span class="px-2 py-1 text-xs rounded ${peer.last_session_status === 'Connected' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}">${peer.last_session_status}</span>` + "`" + ` : ''}
                                <span class="text-sm text-gray-600">${peer.session_count} sessions</span>
                                <span class="text-sm text-gray-600">${peer.event_count} events</span>
                                ${peer.goodbye_count > 0 ? ` + "`" + `<span class="text-sm text-orange-600">${peer.goodbye_count} goodbyes</span>` + "`" + ` : ''}
                                ${peer.mesh_count > 0 ? ` + "`" + `<span class="text-sm text-purple-600">${peer.mesh_count} mesh</span>` + "`" + ` : ''}
                            </div>
                        </div>
                        <div class="text-right text-sm text-gray-500 flex-shrink-0">
                            ${peer.has_scores ? ` + "`" + `
                                <div class="text-xs">
                                    <div>Min Score: <span class="${peer.min_peer_score > 0 ? 'text-green-600' : peer.min_peer_score < 0 ? 'text-red-600' : 'text-gray-600'}">${peer.min_peer_score.toFixed(3)}</span></div>
                                    <div>Max Score: <span class="${peer.max_peer_score > 0 ? 'text-green-600' : peer.max_peer_score < 0 ? 'text-red-600' : 'text-gray-600'}">${peer.max_peer_score.toFixed(3)}</span></div>
                                </div>
                            ` + "`" + ` : ` + "`" + `
                                <div class="text-xs text-gray-400">
                                    <div>No score data</div>
                                </div>
                            ` + "`" + `}
                        </div>
                    </div>
                </div>
            ` + "`" + `;
        }

        function renderPagination() {
            const totalPages = Math.ceil(filteredPeers.length / pageSize);
            const startIndex = (currentPage - 1) * pageSize;
            const endIndex = Math.min(startIndex + pageSize, filteredPeers.length);

            document.getElementById('paginationInfo').textContent =
                ` + "`" + `Showing ${startIndex + 1}-${endIndex} of ${filteredPeers.length} peers` + "`" + `;

            if (totalPages <= 1) {
                document.getElementById('paginationControls').innerHTML = '';
                return;
            }

            let controls = '';

            // Previous button
            if (currentPage > 1) {
                controls += ` + "`" + `<button onclick="changePage(${currentPage - 1})" class="px-3 py-2 border border-gray-300 rounded-md text-sm hover:bg-gray-50">Previous</button>` + "`" + `;
            }

            // Page numbers (show up to 5 pages around current)
            const startPage = Math.max(1, currentPage - 2);
            const endPage = Math.min(totalPages, currentPage + 2);

            if (startPage > 1) {
                controls += ` + "`" + `<button onclick="changePage(1)" class="px-3 py-2 border border-gray-300 rounded-md text-sm hover:bg-gray-50">1</button>` + "`" + `;
                if (startPage > 2) controls += '<span class="px-2 text-gray-500">...</span>';
            }

            for (let i = startPage; i <= endPage; i++) {
                const isActive = i === currentPage;
                controls += ` + "`" + `<button onclick="changePage(${i})" class="px-3 py-2 border rounded-md text-sm ${isActive ? 'bg-blue-600 text-white border-blue-600' : 'border-gray-300 hover:bg-gray-50'}">${i}</button>` + "`" + `;
            }

            if (endPage < totalPages) {
                if (endPage < totalPages - 1) controls += '<span class="px-2 text-gray-500">...</span>';
                controls += ` + "`" + `<button onclick="changePage(${totalPages})" class="px-3 py-2 border border-gray-300 rounded-md text-sm hover:bg-gray-50">${totalPages}</button>` + "`" + `;
            }

            // Next button
            if (currentPage < totalPages) {
                controls += ` + "`" + `<button onclick="changePage(${currentPage + 1})" class="px-3 py-2 border border-gray-300 rounded-md text-sm hover:bg-gray-50">Next</button>` + "`" + `;
            }

            document.getElementById('paginationControls').innerHTML = controls;
        }

        function changePage(page) {
            currentPage = page;
            renderPeerList();
        }

        function updateResultsInfo() {
            const total = allPeers.length;
            const filtered = filteredPeers.length;
            let info = ` + "`" + `${total} total peers` + "`" + `;
            if (filtered !== total) {
                info = ` + "`" + `${filtered} of ${total} peers` + "`" + `;
            }
            document.getElementById('resultsInfo').textContent = info;
        }

        async function showPeerDetails(peerId) {
            document.getElementById('peerModal').classList.remove('hidden');
            document.getElementById('modalTitle').textContent = ` + "`" + `Peer: ${peerId.substring(0, 12)}...` + "`" + `;
            document.getElementById('modalContent').innerHTML =
                '<div class="text-center py-8 text-gray-500"><div class="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full mx-auto mb-4"></div>Loading detailed peer data...</div>';

            // Simulate async loading of detailed data
            setTimeout(() => {
                if (typeof reportData !== 'undefined' && reportData.detailedPeers && reportData.detailedPeers[peerId]) {
                    renderPeerDetails(reportData.detailedPeers[peerId]);
                } else {
                    document.getElementById('modalContent').innerHTML =
                        '<div class="text-center py-8 text-red-500">Detailed peer data not found</div>';
                }
            }, 500);
        }

        // Generate HTML for event counts table
        function generateEventCountsHtml(peerId) {
            const eventCounts = reportData.peerEventCounts && reportData.peerEventCounts[peerId];

            if (!eventCounts || Object.keys(eventCounts).length === 0) {
                return '<div class="p-4 text-center text-gray-500">No event data available</div>';
            }

            // Sort events by count (descending)
            const sortedEvents = Object.entries(eventCounts).sort((a, b) => b[1] - a[1]);

            let html = ` + "`" + `
                <table class="min-w-full bg-white border border-gray-200 rounded text-xs">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="px-3 py-2 text-left">Event Type</th>
                            <th class="px-3 py-2 text-left">Count</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-gray-100">
            ` + "`" + `;

            sortedEvents.forEach(([eventType, count]) => {
                html += ` + "`" + `
                    <tr class="hover:bg-gray-50">
                        <td class="px-3 py-2 text-xs text-gray-900"><code>${eventType}</code></td>
                        <td class="px-3 py-2 text-xs text-gray-700">${count.toLocaleString()}</td>
                    </tr>
                ` + "`" + `;
            });

            html += ` + "`" + `
                    </tbody>
                </table>
            ` + "`" + `;

            return html;
        }

        function renderPeerDetails(peerData) {
            // Render the full detailed view with all peer information
            let sessionsHtml = '';
            if (peerData.connection_sessions && peerData.connection_sessions.length > 0) {
                peerData.connection_sessions.forEach((session, sessionIdx) => {
                    const sessionId = ` + "`" + `session-${sessionIdx}` + "`" + `;
                    let timelineEvents = [];

                    if (session.connected_at) timelineEvents.push({type: 'connected', time: session.connected_at, label: 'Connected'});
                    if (session.identified_at) timelineEvents.push({type: 'identified', time: session.identified_at, label: 'Identified'});
                    if (session.mesh_events) {
                        session.mesh_events.forEach(event => {
                            timelineEvents.push({type: 'mesh', time: event.timestamp, label: ` + "`" + `${event.type}: ${event.topic}` + "`" + `});
                        });
                    }
                    if (session.goodbye_events) {
                        session.goodbye_events.forEach(event => {
                            timelineEvents.push({type: 'goodbye', time: event.timestamp, label: ` + "`" + `Goodbye: ${event.reason}` + "`" + `});
                        });
                    }
                    if (session.disconnected_at) timelineEvents.push({type: 'disconnected', time: session.disconnected_at, label: 'Disconnected'});

                    timelineEvents.sort((a, b) => new Date(a.time) - new Date(b.time));

                    const timelineHtml = timelineEvents.map(event => {
                        const color = event.type === 'connected' ? 'green' :
                                     event.type === 'identified' ? 'blue' :
                                     event.type === 'mesh' ? 'purple' :
                                     event.type === 'goodbye' ? 'orange' : 'red';
                        return ` + "`" + `
                            <tr class="hover:bg-gray-50">
                                <td class="px-3 py-2 text-xs">${new Date(event.time).toLocaleTimeString()}</td>
                                <td class="px-3 py-2 text-xs">
                                    <span class="px-2 py-1 text-xs bg-${color}-100 text-${color}-800 rounded">${event.type.toUpperCase()}</span>
                                </td>
                                <td class="px-3 py-2 text-xs text-gray-700">${event.label}</td>
                            </tr>
                        ` + "`" + `;
                    }).join('');

                    const scoreSnapshotsHtml = session.peer_scores ? session.peer_scores.map((snapshot, idx) => {
                        const rowId = ` + "`" + `${sessionId}-score-${idx}` + "`" + `;
                        const topicsHtml = snapshot.topics && snapshot.topics.length > 0 ?
                            snapshot.topics.map(topic => ` + "`" + `
                                <div class="mb-2 p-2 bg-gray-50 rounded text-xs">
                                    <div class="font-medium text-gray-700 mb-1">Topic: ${topic.topic}</div>
                                    <div class="grid grid-cols-2 gap-2 text-xs">
                                        <div>Time in Mesh: ${(topic.time_in_mesh / 1000000000).toFixed(1)}s</div>
                                        <div>First Deliveries: ${topic.first_message_deliveries.toFixed(3)}</div>
                                        <div>Mesh Deliveries: ${topic.mesh_message_deliveries.toFixed(3)}</div>
                                        <div>Invalid Deliveries: ${topic.invalid_message_deliveries.toFixed(3)}</div>
                                    </div>
                                </div>
                            ` + "`" + `).join('') : '<div class="text-gray-500 text-xs p-2">No topic data available</div>';

                        return ` + "`" + `
                            <tr class="hover:bg-gray-50">
                                <td class="px-3 py-2 text-xs">${new Date(snapshot.timestamp).toLocaleTimeString()}</td>
                                <td class="px-3 py-2 text-xs font-medium ${snapshot.score > 0 ? 'text-green-600' : snapshot.score < 0 ? 'text-red-600' : 'text-gray-600'}">${snapshot.score.toFixed(3)}</td>
                                <td class="px-3 py-2 text-xs">${snapshot.app_specific_score.toFixed(3)}</td>
                                <td class="px-3 py-2 text-xs">${snapshot.ip_colocation_factor.toFixed(3)}</td>
                                <td class="px-3 py-2 text-xs">${snapshot.behaviour_penalty.toFixed(3)}</td>
                                <td class="px-3 py-2 text-xs">
                                    ${snapshot.topics && snapshot.topics.length > 0 ?
                                        ` + "`" + `<button class="text-blue-600 hover:text-blue-800 underline cursor-pointer" onclick="toggleSection('${rowId}')">${snapshot.topics.length} topics</button>` + "`" + ` :
                                        'None'
                                    }
                                </td>
                            </tr>
                            ${snapshot.topics && snapshot.topics.length > 0 ? ` + "`" + `
                            <tr id="${rowId}" class="hidden">
                                <td colspan="6" class="px-3 py-2 bg-blue-50">
                                    <div class="max-h-48 overflow-y-auto">
                                        ${topicsHtml}
                                    </div>
                                </td>
                            </tr>
                            ` + "`" + ` : ''}
                        ` + "`" + `;
                    }).join('') : '<tr><td colspan="6" class="text-center py-4 text-gray-500">No score data</td></tr>';

                    sessionsHtml += ` + "`" + `
                        <div class="border border-gray-200 rounded-lg mb-4">
                            <div class="p-3 bg-gray-50 cursor-pointer" onclick="toggleSection('${sessionId}')">
                                <div class="flex items-center justify-between">
                                    <div class="flex items-center space-x-4">
                                        <span class="font-medium text-gray-900">Session ${sessionIdx + 1}</span>
                                        <span class="text-sm text-gray-600">${session.connection_duration ? (session.connection_duration / 1000000000).toFixed(2) + 's' : 'Unknown duration'}</span>
                                        <span class="text-sm text-gray-600">${session.message_count || 0} messages</span>
                                        ${session.peer_scores ? ` + "`" + `<span class="text-sm text-gray-600">${session.peer_scores.length} score snapshots</span>` + "`" + ` : ''}
                                        ${session.goodbye_events && session.goodbye_events.length > 0 ? ` + "`" + `<span class="text-sm text-orange-600">${session.goodbye_events.length} goodbye events</span>` + "`" + ` : ''}
                                        ${session.mesh_events && session.mesh_events.length > 0 ? ` + "`" + `<span class="text-sm text-purple-600">${session.mesh_events.length} mesh events</span>` + "`" + ` : ''}
                                        <span class="px-2 py-1 text-xs ${session.disconnected ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'} rounded">
                                            ${session.disconnected ? 'Disconnected' : 'Connected'}
                                        </span>
                                    </div>
                                    <svg class="w-4 h-4 text-gray-500 transform transition-transform" id="${sessionId}-arrow">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                    </svg>
                                </div>
                            </div>
                            <div class="hidden p-4 border-t border-gray-200" id="${sessionId}">
                                <div class="space-y-4">
                                    ${session.peer_scores ? ` + "`" + `
                                    <div>
                                        <div class="p-3 bg-gray-50 cursor-pointer border rounded-lg" onclick="toggleSection('${sessionId}-scores')">
                                            <div class="flex items-center justify-between">
                                                <h6 class="font-medium text-gray-800">Peer Score Evolution (${session.peer_scores.length} snapshots)</h6>
                                                <svg class="w-4 h-4 text-gray-500 transform transition-transform" id="${sessionId}-scores-arrow">
                                                    <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                                </svg>
                                            </div>
                                        </div>
                                        <div class="hidden mt-2" id="${sessionId}-scores">
                                            <div class="max-h-64 overflow-y-auto">
                                                <table class="min-w-full bg-white border border-gray-200 rounded text-xs">
                                                    <thead class="bg-gray-50">
                                                        <tr>
                                                            <th class="px-3 py-2 text-left">Time</th>
                                                            <th class="px-3 py-2 text-left">Total Score</th>
                                                            <th class="px-3 py-2 text-left">App Score</th>
                                                            <th class="px-3 py-2 text-left">IP Colocation</th>
                                                            <th class="px-3 py-2 text-left">Behaviour</th>
                                                            <th class="px-3 py-2 text-left">Topics</th>
                                                        </tr>
                                                    </thead>
                                                    <tbody class="divide-y divide-gray-100">
                                                        ${scoreSnapshotsHtml}
                                                    </tbody>
                                                </table>
                                            </div>
                                        </div>
                                    </div>
                                    ` + "`" + ` : ''}
                                    <div>
                                        <div class="p-3 bg-gray-50 cursor-pointer border rounded-lg" onclick="toggleSection('${sessionId}-timeline')">
                                            <div class="flex items-center justify-between">
                                                <h6 class="font-medium text-gray-800">Session Timeline</h6>
                                                <svg class="w-4 h-4 text-gray-500 transform transition-transform" id="${sessionId}-timeline-arrow">
                                                    <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                                </svg>
                                            </div>
                                        </div>
                                        <div class="hidden mt-2" id="${sessionId}-timeline">
                                            <div class="max-h-64 overflow-y-auto">
                                                <table class="min-w-full bg-white border border-gray-200 rounded text-xs">
                                                    <thead class="bg-gray-50">
                                                        <tr>
                                                            <th class="px-3 py-2 text-left">Time</th>
                                                            <th class="px-3 py-2 text-left">Event Type</th>
                                                            <th class="px-3 py-2 text-left">Details</th>
                                                        </tr>
                                                    </thead>
                                                    <tbody class="divide-y divide-gray-100">
                                                        ${timelineHtml}
                                                    </tbody>
                                                </table>
                                            </div>
                                        </div>
                                    </div>

                                    <!-- Mesh Events Section -->
                                    ${session.mesh_events && session.mesh_events.length > 0 ? ` + "`" + `
                                    <div>
                                        <div class="p-3 bg-gray-50 cursor-pointer border rounded-lg" onclick="toggleSection('${sessionId}-mesh')">
                                            <div class="flex items-center justify-between">
                                                <h6 class="font-medium text-gray-800">Mesh Participation Events (${session.mesh_events.length} events)</h6>
                                                <svg class="w-4 h-4 text-gray-500 transform transition-transform" id="${sessionId}-mesh-arrow">
                                                    <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                                </svg>
                                            </div>
                                        </div>
                                        <div class="hidden mt-2" id="${sessionId}-mesh">
                                            <div class="max-h-64 overflow-y-auto">
                                                <table class="min-w-full bg-white border border-gray-200 rounded text-xs">
                                                    <thead class="bg-gray-50">
                                                        <tr>
                                                            <th class="px-3 py-2 text-left">Time</th>
                                                            <th class="px-3 py-2 text-left">Type</th>
                                                            <th class="px-3 py-2 text-left">Direction</th>
                                                            <th class="px-3 py-2 text-left">Topic</th>
                                                            <th class="px-3 py-2 text-left">Reason</th>
                                                        </tr>
                                                    </thead>
                                                    <tbody class="divide-y divide-gray-100">
                                                        ${session.mesh_events.map(meshEvent => ` + "`" + `
                                                            <tr class="hover:bg-gray-50">
                                                                <td class="px-3 py-2 text-xs">${new Date(meshEvent.timestamp).toLocaleTimeString()}</td>
                                                                <td class="px-3 py-2 text-xs">
                                                                    <span class="px-2 py-1 text-xs rounded ${meshEvent.type === 'GRAFT' ? 'bg-green-100 text-green-800' : meshEvent.type === 'PRUNE' ? 'bg-red-100 text-red-800' : 'bg-gray-100 text-gray-800'}">
                                                                        ${meshEvent.type}
                                                                    </span>
                                                                </td>
                                                                <td class="px-3 py-2 text-xs text-gray-700">${meshEvent.direction || '-'}</td>
                                                                <td class="px-3 py-2 text-xs">
                                                                    <span class="font-mono text-xs bg-gray-100 px-2 py-1 rounded">${meshEvent.topic}</span>
                                                                </td>
                                                                <td class="px-3 py-2 text-xs text-gray-600">${meshEvent.reason || '-'}</td>
                                                            </tr>
                                                        ` + "`" + `).join('')}
                                                    </tbody>
                                                </table>
                                            </div>
                                        </div>
                                    </div>
                                    ` + "`" + ` : ''}
                                </div>
                            </div>
                        </div>
                    ` + "`" + `;
                });
            }

            const clientInfo = getClientLogo(peerData.client_type);
            const logoImg = clientInfo ?
                ` + "`" + `<img src="${clientInfo.logo}" alt="${clientInfo.displayName}" class="w-12 h-12 rounded-md object-cover client-logo" onerror="this.style.display='none'">` + "`" + ` :
                ` + "`" + `<div class="w-12 h-12 rounded-md flex items-center justify-center text-white text-lg client-fallback">${peerData.client_type.substring(0, 2).toUpperCase()}</div>` + "`" + `;

            document.getElementById('modalContent').innerHTML = ` + "`" + `
                <div class="space-y-6">
                    <!-- Client Header -->
                    <div class="flex items-center space-x-4 p-4 bg-gradient-to-r from-blue-50 to-indigo-50 rounded-lg border border-blue-200">
                        <div class="flex-shrink-0">
                            ${logoImg}
                        </div>
                        <div class="flex-1">
                            <div class="flex items-center space-x-3">
                                <h3 class="text-lg font-semibold text-gray-900">${clientInfo ? clientInfo.displayName : peerData.client_type}</h3>
                            </div>
                            <p class="text-sm text-gray-600 mt-1">${peerData.client_agent}</p>
                        </div>
                    </div>

                    <!-- Basic Information -->
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div>
                            <div class="text-sm font-medium text-gray-500">Full Peer ID</div>
                            <div class="text-sm font-mono break-all">${peerData.peer_id}</div>
                        </div>
                        <div>
                            <div class="text-sm font-medium text-gray-500">Total Sessions</div>
                            <div class="text-sm">${peerData.connection_sessions ? peerData.connection_sessions.length : 0}</div>
                        </div>
                        ${peerData.first_seen_at ? ` + "`" + `
                        <div>
                            <div class="text-sm font-medium text-gray-500">First Seen</div>
                            <div class="text-sm">${new Date(peerData.first_seen_at).toLocaleString()}</div>
                        </div>
                        ` + "`" + ` : ''}
                        ${peerData.last_seen_at ? ` + "`" + `
                        <div>
                            <div class="text-sm font-medium text-gray-500">Last Seen</div>
                            <div class="text-sm">${new Date(peerData.last_seen_at).toLocaleString()}</div>
                        </div>
                        ` + "`" + ` : ''}
                    </div>

                    <!-- Connection Sessions -->
                    <div>
                        ${sessionsHtml || '<div class="text-center py-8 text-gray-500">No session data available</div>'}
                    </div>

                    <!-- Event Counts -->
                    <div>
                        <div class="p-3 bg-gray-50 cursor-pointer border rounded-lg" onclick="toggleSection('peer-events-${peerData.peer_id}')">
                            <div class="flex items-center justify-between">
                                <h5 class="font-medium text-gray-900">Event Counts</h5>
                                <svg class="w-4 h-4 text-gray-500 transform transition-transform" id="peer-events-${peerData.peer_id}-arrow">
                                    <path stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                                </svg>
                            </div>
                        </div>
                        <div class="hidden mt-2" id="peer-events-${peerData.peer_id}">
                            <div class="max-h-64 overflow-y-auto">
                                ${generateEventCountsHtml(peerData.peer_id)}
                            </div>
                        </div>
                    </div>
                </div>
            ` + "`" + `;
        }

        function toggleSection(sectionId) {
            const content = document.getElementById(sectionId);
            const arrow = document.getElementById(sectionId + '-arrow');
            const toggle = document.getElementById(sectionId + '-toggle');

            if (content.classList.contains('hidden')) {
                content.classList.remove('hidden');
                if (arrow) arrow.style.transform = 'rotate(180deg)';
                if (toggle) toggle.textContent = sectionId === 'ai-analysis' ? 'Hide Analysis' : 'Hide';
            } else {
                content.classList.add('hidden');
                if (arrow) arrow.style.transform = 'rotate(0deg)';
                if (toggle) toggle.textContent = sectionId === 'ai-analysis' ? 'Show Analysis' : 'Show';
            }
        }

        function closePeerModal() {
            document.getElementById('peerModal').classList.add('hidden');
        }

        // AI Analysis Modal functions
        function openAIAnalysisModal() {
            document.getElementById('aiAnalysisModal').classList.remove('hidden');
        }

        function closeAIAnalysisModal() {
            document.getElementById('aiAnalysisModal').classList.add('hidden');
        }

        function exportFilteredData() {
            const exportData = {
                summary: {
                    total_peers: allPeers.length,
                    filtered_peers: filteredPeers.length,
                    filters_applied: {
                        search: document.getElementById('search').value,
                        sort_by: sortBy
                    }
                },
                peers: filteredPeers
            };

            const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = ` + "`" + `hermes-peer-score-filtered-${new Date().toISOString().split('T')[0]}.json` + "`" + `;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(url);
        }

        // Close modals when clicking outside
        document.getElementById('peerModal').addEventListener('click', function(e) {
            if (e.target === this) {
                closePeerModal();
            }
        });

        // Close AI analysis modal when clicking outside (if it exists)
        const aiModal = document.getElementById('aiAnalysisModal');
        if (aiModal) {
            aiModal.addEventListener('click', function(e) {
                if (e.target === this) {
                    closeAIAnalysisModal();
                }
            });
        }
    </script>
</body>
</html>`

// Helper functions for the optimized report generation

// extractSummaryData creates a summary from the full report data.
func extractSummaryData(report PeerScoreReport) SummaryData {
	summary := SummaryData{
		TestDuration:         report.Duration.Seconds(),
		StartTime:            report.StartTime,
		EndTime:              report.EndTime,
		TotalConnections:     report.TotalConnections,
		SuccessfulHandshakes: report.SuccessfulHandshakes,
		FailedHandshakes:     report.FailedHandshakes,
		UniquePeers:          len(report.Peers),
		PeerSummaries:        make([]PeerSummary, 0, len(report.Peers)),
	}

	// Create peer summaries sorted by event count
	for peerID, peer := range report.Peers {
		eventCount := 0

		if events, exists := report.PeerEventCounts[peerID]; exists {
			for _, count := range events {
				eventCount += count
			}
		}

		var (
			goodbyeCount       = 0
			meshCount          = 0
			minScore, maxScore float64
			hasScores          = false
			scoreInitialized   = false
		)

		for _, session := range peer.ConnectionSessions {
			goodbyeCount += len(session.GoodbyeEvents)
			meshCount += len(session.MeshEvents)

			// Find min/max peer scores across all sessions
			for _, scoreSnapshot := range session.PeerScores {
				if !scoreInitialized {
					minScore = scoreSnapshot.Score
					maxScore = scoreSnapshot.Score
					scoreInitialized = true
					hasScores = true
				} else {
					if scoreSnapshot.Score < minScore {
						minScore = scoreSnapshot.Score
					}

					if scoreSnapshot.Score > maxScore {
						maxScore = scoreSnapshot.Score
					}
				}
			}
		}

		shortPeerID := peerID
		if len(peerID) > 12 {
			shortPeerID = peerID[:12]
		}

		// Determine last session status and time
		var (
			lastSessionStatus string
			lastSessionTime   string
		)

		if len(peer.ConnectionSessions) > 0 {
			// Find the most recent session (by connected_at time)
			var (
				mostRecentSession *ConnectionSession
				mostRecentTime    time.Time
			)

			for i := range peer.ConnectionSessions {
				session := &peer.ConnectionSessions[i]
				if session.ConnectedAt != nil && (mostRecentSession == nil || session.ConnectedAt.After(mostRecentTime)) {
					mostRecentSession = session
					mostRecentTime = *session.ConnectedAt
				}
			}

			if mostRecentSession != nil {
				if mostRecentSession.Disconnected {
					lastSessionStatus = "Disconnected"

					if mostRecentSession.DisconnectedAt != nil {
						lastSessionTime = mostRecentSession.DisconnectedAt.Format("15:04:05")
					}
				} else {
					lastSessionStatus = "Connected"

					if mostRecentSession.ConnectedAt != nil {
						lastSessionTime = mostRecentSession.ConnectedAt.Format("15:04:05")
					}
				}
			}
		}

		summary.PeerSummaries = append(summary.PeerSummaries, PeerSummary{
			PeerID:            peerID,
			ShortPeerID:       shortPeerID,
			ClientType:        peer.ClientType,
			ClientAgent:       peer.ClientAgent,
			SessionCount:      len(peer.ConnectionSessions),
			EventCount:        eventCount,
			GoodbyeCount:      goodbyeCount,
			MeshCount:         meshCount,
			MinPeerScore:      minScore,
			MaxPeerScore:      maxScore,
			HasScores:         hasScores,
			LastSessionStatus: lastSessionStatus,
			LastSessionTime:   lastSessionTime,
		})
	}

	return summary
}

// cleanAIHTML sanitizes AI-generated HTML content to prevent JavaScript errors.
func cleanAIHTML(content string) string {
	if content == "" {
		return content
	}

	// Remove any null bytes or other control characters that could break JavaScript.
	re := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F\x7F]`)
	content = re.ReplaceAllString(content, "")

	// Replace any problematic characters that could break JSON/JavaScript.
	content = strings.ReplaceAll(content, "\u0000", "")
	content = strings.ReplaceAll(content, "\ufffd", "") // Replacement character

	// Escape Go template syntax that AI might accidentally generate
	// This is critical because {{}} syntax would break the Go template parsing
	content = strings.ReplaceAll(content, "{{", "&#123;&#123;")
	content = strings.ReplaceAll(content, "}}", "&#125;&#125;")

	// Escape other problematic template-like syntax
	content = strings.ReplaceAll(content, "${", "&#36;&#123;")

	// Trim any trailing whitespace
	content = strings.TrimSpace(content)

	return content
}

// generateDataFile creates a JavaScript file containing the report data.
func generateDataFile(log logrus.FieldLogger, report PeerScoreReport, dataFile string) error {
	// Clean all peer data to ensure no invalid characters
	cleanedPeers := make(map[string]*PeerStats)

	for peerID, peer := range report.Peers {
		cleanedPeer := *peer
		// Clean client agent strings that might contain invalid characters
		cleanedPeer.ClientAgent = cleanAIHTML(peer.ClientAgent)
		cleanedPeer.ClientType = cleanAIHTML(peer.ClientType)
		cleanedPeers[peerID] = &cleanedPeer
	}

	// Create the data structure that will be embedded in the JS file
	data := map[string]interface{}{
		"peers":           extractSummaryData(report).PeerSummaries,
		"detailedPeers":   cleanedPeers,           // Full detailed data for on-demand loading
		"peerEventCounts": report.PeerEventCounts, // Event counts by peer ID
		"summary":         extractSummaryData(report),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Validate that the JSON is valid by unmarshaling it
	var testData interface{}
	if err := json.Unmarshal(jsonData, &testData); err != nil {
		return fmt.Errorf("generated JSON is invalid: %w", err)
	}

	// Create JavaScript file with embedded data
	jsContent := fmt.Sprintf("const reportData = %s;", string(jsonData))

	// Validate that the JavaScript content looks reasonable
	if len(jsContent) < 50 {
		return fmt.Errorf("generated JavaScript content is suspiciously small: %d bytes", len(jsContent))
	}

	log.Printf("Writing data file: %s (size: %d bytes)\n", dataFile, len(jsContent))

	//nolint:gosec // controlled.
	if err := os.WriteFile(dataFile, []byte(jsContent), 0644); err != nil {
		return fmt.Errorf("failed to write data file: %w", err)
	}

	log.Printf("Data file written successfully\n")

	return nil
}

// GenerateHTMLReport creates an optimized HTML report from a JSON report file.
// It generates a report that loads data dynamically to avoid browser lockup with large datasets.
//
// Parameters:
//   - jsonFile: Path to the input JSON report file
//   - outputFile: Path where the HTML report should be written
//
// Returns an error if file operations or template processing fails.
func GenerateHTMLReport(log logrus.FieldLogger, jsonFile, outputFile string) error {
	return GenerateHTMLReportWithAI(log, jsonFile, outputFile, "", "")
}

// GenerateHTMLReportWithAI creates an optimized HTML report with optional AI analysis.
func GenerateHTMLReportWithAI(log logrus.FieldLogger, jsonFile, outputFile, apiKey, aiAnalysis string) error {
	// Read the JSON report file from disk.
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	// Parse the JSON data into our report structure.
	var report PeerScoreReport
	if uErr := json.Unmarshal(data, &report); uErr != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", uErr)
	}

	// Generate AI analysis if API key is provided and analysis is not pre-generated
	var finalAIAnalysis string
	if aiAnalysis != "" {
		finalAIAnalysis = aiAnalysis
	} else if apiKey != "" {
		log.Printf("Generating AI analysis...")

		summary := SummarizeReport(&report)

		// Check summary size and log it
		summaryJSON, _ := json.Marshal(summary)
		summarySize := len(summaryJSON)

		log.Printf("Summary data size: %d bytes (%d KB)", summarySize, summarySize/1024)

		log.Printf("Creating AI client and sending analysis request...")

		client := NewClaudeAPIClient(apiKey)

		analysis, aerr := client.AnalyzeReport(log, summary)
		if aerr != nil {
			log.Printf("Warning: Failed to generate AI analysis: %v", aerr)

			finalAIAnalysis = "⚠️ AI analysis failed to generate. Large dataset may have caused timeout. Please try with a smaller report or check your API connection."
		} else {
			// Clean the AI-generated content to prevent JavaScript errors
			finalAIAnalysis = cleanAIHTML(analysis)

			log.Printf("AI analysis generated successfully")
		}
	}

	// Generate the data file alongside the HTML report
	dataFile := strings.Replace(outputFile, ".html", "-data.js", 1)
	if gerr := generateDataFile(log, report, dataFile); gerr != nil {
		return fmt.Errorf("failed to generate data file: %w", gerr)
	}

	// Prepare template data with summary information only
	templateData := OptimizedHTMLTemplateData{
		GeneratedAt:      time.Now(),
		Summary:          extractSummaryData(report),
		DataFile:         filepath.Base(dataFile),
		AIAnalysis:       finalAIAnalysis,
		AIAnalysisHTML:   template.HTML(finalAIAnalysis), //nolint:gosec // data sanitized further down.
		ValidationMode:   report.ValidationMode,
		ValidationConfig: report.ValidationConfig,
	}

	// Create the optimized HTML template with custom functions
	tmpl := template.New("report").Funcs(template.FuncMap{
		"mul": func(a, b interface{}) float64 {
			switch va := a.(type) {
			case float64:
				if vb, ok := b.(float64); ok {
					return va * vb
				}
			case int:
				if vb, ok := b.(int); ok {
					return float64(va * vb)
				}
				if vb, ok := b.(float64); ok {
					return float64(va) * vb
				}
			}

			return 0
		},
		"div": func(a, b interface{}) float64 {
			switch va := a.(type) {
			case float64:
				if vb, ok := b.(float64); ok && vb != 0 {
					return va / vb
				}
			case int:
				if vb, ok := b.(int); ok && vb != 0 {
					return float64(va) / float64(vb)
				}
				if vb, ok := b.(float64); ok && vb != 0 {
					return float64(va) / vb
				}
			}

			return 0
		},
	})

	// Parse the optimized HTML template string
	tmpl, err = tmpl.Parse(optimizedHTMLTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Ensure the output directory exists
	if mkErr := os.MkdirAll(filepath.Dir(outputFile), 0755); mkErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkErr)
	}

	// Create the output HTML file
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute the template with summary data
	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	log.Printf("Optimized HTML report generated: %s", outputFile)
	log.Printf("Data file generated: %s", dataFile)

	return nil
}
