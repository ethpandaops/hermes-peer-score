// Package main provides HTML report generation functionality for the Hermes peer score tool.
// This file handles the conversion of JSON reports into styled HTML presentations suitable
// for web viewing and GitHub Pages deployment.

package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"time"
)

// htmlTemplate contains the complete HTML template for generating peer score reports.
// It uses Tailwind CSS for styling and includes responsive design elements.
// The template includes sections for overall score, statistics grid, client distribution,
// goodbye message analysis, and detailed test information.
const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Hermes Peer Score Report</title>
    <script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <div class="container mx-auto px-4 py-8">
        <!-- Header -->
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <div class="flex items-center justify-between">
                <div>
                    <h1 class="text-3xl font-bold text-gray-900">Hermes Peer Score Report</h1>
                </div>
                <div class="text-right">
                    <p class="text-sm text-gray-500">Generated on</p>
                    <p class="text-lg font-semibold">{{.GeneratedAt.Format "2006-01-02 15:04:05 UTC"}}</p>
                </div>
            </div>
        </div>

        <!-- Overall Score -->
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <div class="text-center">
                <h2 class="text-2xl font-bold mb-4">Overall Score</h2>
                <div class="inline-flex items-center justify-center w-32 h-32 rounded-full {{if lt .Report.OverallScore 50.0}}bg-red-100{{else if lt .Report.OverallScore 80.0}}bg-yellow-100{{else}}bg-green-100{{end}}">
                    <span class="text-4xl font-bold {{if lt .Report.OverallScore 50.0}}text-red-600{{else if lt .Report.OverallScore 80.0}}text-yellow-600{{else}}text-green-600{{end}}">
                        {{printf "%.1f" .Report.OverallScore}}%
                    </span>
                </div>
                <p class="mt-4 text-lg font-semibold {{if lt .Report.OverallScore 50.0}}text-red-600{{else if lt .Report.OverallScore 80.0}}text-yellow-600{{else}}text-green-600{{end}}">
                    {{.ScoreClassification}}
                </p>
                <p class="text-gray-600 mt-2">{{.Report.Summary}}</p>
            </div>
        </div>

        <!-- Statistics Grid -->
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
            <!-- Test Duration -->
            <div class="bg-white rounded-lg shadow p-6">
                <div class="flex items-center">
                    <div class="p-3 rounded-full bg-blue-100">
                        <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                    </div>
                    <div class="ml-4">
                        <p class="text-2xl font-semibold text-gray-900">{{.Report.Duration.Round (time.Second)}}</p>
                        <p class="text-gray-600">Test Duration</p>
                    </div>
                </div>
            </div>

            <!-- Total Connections -->
            <div class="bg-white rounded-lg shadow p-6">
                <div class="flex items-center">
                    <div class="p-3 rounded-full bg-green-100">
                        <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
                        </svg>
                    </div>
                    <div class="ml-4">
                        <p class="text-2xl font-semibold text-gray-900">{{.Report.TotalConnections}}</p>
                        <p class="text-gray-600">Total Connections</p>
                    </div>
                </div>
            </div>

            <!-- Successful Handshakes -->
            <div class="bg-white rounded-lg shadow p-6">
                <div class="flex items-center">
                    <div class="p-3 rounded-full bg-purple-100">
                        <svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                        </svg>
                    </div>
                    <div class="ml-4">
                        <p class="text-2xl font-semibold text-gray-900">{{.Report.SuccessfulHandshakes}}</p>
                        <p class="text-gray-600">Successful Handshakes</p>
                        <p class="text-sm text-gray-500">({{printf "%.1f" .ConnectionRate}}% success rate)</p>
                    </div>
                </div>
            </div>

            <!-- Unique Clients -->
            <div class="bg-white rounded-lg shadow p-6">
                <div class="flex items-center">
                    <div class="p-3 rounded-full bg-indigo-100">
                        <svg class="w-6 h-6 text-indigo-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 515.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"></path>
                        </svg>
                    </div>
                    <div class="ml-4">
                        <p class="text-2xl font-semibold text-gray-900">{{.Report.UniqueClients}}</p>
                        <p class="text-gray-600">Unique Clients</p>
                    </div>
                </div>
            </div>

        </div>

        <!-- Client Distribution -->
        {{if .ClientList}}
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <h3 class="text-xl font-bold mb-4">Client Distribution</h3>
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                {{range .ClientList}}
                <div class="bg-gray-50 rounded-lg p-4 flex justify-between items-center">
                    <span class="font-medium capitalize">{{.Name}}</span>
                    <span class="bg-blue-100 text-blue-800 px-3 py-1 rounded-full text-sm font-semibold">{{.Count}}</span>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        <!-- Goodbye Messages -->
        {{if .Report.GoodbyeReasons}}
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <h3 class="text-xl font-bold mb-4">Goodbye Messages ({{.Report.GoodbyeMessages}} total)</h3>

            <!-- Overall breakdown -->
            <div class="mb-6">
                <h4 class="font-semibold text-gray-700 mb-3">Overall Breakdown</h4>
                <div class="space-y-2">
                    {{range $reason, $count := .Report.GoodbyeReasons}}
                    <div class="flex justify-between items-center p-3 bg-gray-50 rounded-lg">
                        <span class="text-gray-700">{{$reason}}</span>
                        <span class="bg-gray-200 text-gray-800 px-3 py-1 rounded-full text-sm font-semibold">{{$count}}</span>
                    </div>
                    {{end}}
                </div>
            </div>

            <!-- By client breakdown -->
            {{if .Report.GoodbyesByClient}}
            <div>
                <h4 class="font-semibold text-gray-700 mb-3">By Client Type</h4>
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {{range $client, $reasons := .Report.GoodbyesByClient}}
                    <div class="bg-gray-50 rounded-lg p-4">
                        <h5 class="font-medium capitalize text-gray-800 mb-2">{{$client}}</h5>
                        <div class="space-y-1">
                            {{range $reason, $count := $reasons}}
                            <div class="flex justify-between text-sm">
                                <span class="text-gray-600">{{$reason}}</span>
                                <span class="font-medium">{{$count}}</span>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
        {{end}}

        <!-- Timing Analysis -->
        {{if gt .Report.TimingAnalysis.TotalConnections 0}}
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <h3 class="text-xl font-bold mb-4">Timing Analysis</h3>

            <!-- Overall timing statistics -->
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
                <div class="bg-gray-50 rounded-lg p-4">
                    <h4 class="font-medium text-gray-700 mb-1">Average Duration</h4>
                    <p class="text-lg font-semibold">{{.Report.TimingAnalysis.AverageConnectionDuration.Round (time.Second)}}</p>
                </div>
                <div class="bg-gray-50 rounded-lg p-4">
                    <h4 class="font-medium text-gray-700 mb-1">Median Duration</h4>
                    <p class="text-lg font-semibold">{{.Report.TimingAnalysis.MedianConnectionDuration.Round (time.Second)}}</p>
                </div>
                <div class="bg-gray-50 rounded-lg p-4">
                    <h4 class="font-medium text-gray-700 mb-1">Fastest Disconnect</h4>
                    <p class="text-lg font-semibold">{{.Report.TimingAnalysis.FastestDisconnect.Round (time.Second)}}</p>
                </div>
                <div class="bg-gray-50 rounded-lg p-4">
                    <h4 class="font-medium text-gray-700 mb-1">Longest Connection</h4>
                    <p class="text-lg font-semibold">{{.Report.TimingAnalysis.LongestConnection.Round (time.Second)}}</p>
                </div>
            </div>

            <!-- Goodbye reason timings -->
            {{if .Report.TimingAnalysis.GoodbyeReasonTimings}}
            <div class="mb-6">
                <h4 class="font-semibold text-gray-700 mb-3">Goodbye Timing Analysis (Overall)</h4>
                <div class="overflow-x-auto">
                    <table class="min-w-full bg-white border border-gray-200 rounded-lg">
                        <thead class="bg-gray-50">
                            <tr>
                                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Reason</th>
                                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Count</th>
                                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Avg Time</th>
                                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Median</th>
                                <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Min/Max</th>
                            </tr>
                        </thead>
                        <tbody class="bg-white divide-y divide-gray-200">
                            {{range $reason, $timing := .Report.TimingAnalysis.GoodbyeReasonTimings}}
                            <tr class="hover:bg-gray-50">
                                <td class="px-4 py-2 text-sm text-gray-900">{{$reason}}</td>
                                <td class="px-4 py-2 text-sm text-gray-900">{{$timing.Count}}</td>
                                <td class="px-4 py-2 text-sm text-gray-900">{{$timing.AverageDuration.Round (time.Second)}}</td>
                                <td class="px-4 py-2 text-sm text-gray-900">{{$timing.MedianDuration.Round (time.Second)}}</td>
                                <td class="px-4 py-2 text-sm text-gray-500">{{$timing.MinDuration.Round (time.Second)}} / {{$timing.MaxDuration.Round (time.Second)}}</td>
                            </tr>
                            {{end}}
                        </tbody>
                    </table>
                </div>
            </div>

            <!-- Goodbye timing by client -->
            <div class="mb-6">
                <h4 class="font-semibold text-gray-700 mb-3">Goodbye Timing Analysis by Client</h4>
                {{range $client, $reasons := .Report.GoodbyesByClient}}
                <div class="mb-4">
                    <h5 class="font-medium text-gray-600 mb-2 capitalize">{{$client}}</h5>
                    <div class="overflow-x-auto">
                        <table class="min-w-full bg-white border border-gray-200 rounded-lg">
                            <thead class="bg-gray-50">
                                <tr>
                                    <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Reason</th>
                                    <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Count</th>
                                    <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Avg Time</th>
                                    <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Median</th>
                                    <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Min/Max</th>
                                </tr>
                            </thead>
                            <tbody class="bg-white divide-y divide-gray-200">
                                {{range $reason, $count := $reasons}}
                                {{$clientTimings := index $.Report.TimingAnalysis.ClientSpecificTimings $client}}
                                {{if $clientTimings}}
                                {{$clientReasonTiming := index $clientTimings $reason}}
                                {{if $clientReasonTiming}}
                                <tr class="hover:bg-gray-50">
                                    <td class="px-4 py-2 text-sm text-gray-900">{{$reason}}</td>
                                    <td class="px-4 py-2 text-sm text-gray-900">{{$clientReasonTiming.Count}}</td>
                                    <td class="px-4 py-2 text-sm text-gray-900">{{$clientReasonTiming.AverageDuration.Round (time.Second)}}</td>
                                    <td class="px-4 py-2 text-sm text-gray-900">{{$clientReasonTiming.MedianDuration.Round (time.Second)}}</td>
                                    <td class="px-4 py-2 text-sm text-gray-500">{{$clientReasonTiming.MinDuration.Round (time.Second)}} / {{$clientReasonTiming.MaxDuration.Round (time.Second)}}</td>
                                </tr>
                                {{end}}
                                {{end}}
                                {{end}}
                            </tbody>
                        </table>
                    </div>
                </div>
                {{end}}
            </div>

            {{end}}

            <!-- Timing patterns -->
            {{if .Report.TimingAnalysis.ClientTimingPatterns}}
            <div class="mb-6">
                <h4 class="font-semibold text-gray-700 mb-3">Detected Timing Patterns</h4>
                <div class="space-y-3">
                    {{range .Report.TimingAnalysis.ClientTimingPatterns}}
                    <div class="bg-blue-50 border border-blue-200 rounded-lg p-4">
                        <div class="flex justify-between items-start">
                            <div class="flex-1">
                                <p class="font-medium text-blue-900">{{.ClientType}} - {{.GoodbyeReason}}</p>
                                <p class="text-sm text-blue-700 mt-1">{{.Pattern}}</p>
                            </div>
                            <div class="text-right">
                                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                                    {{.Occurrences}} occurrences
                                </span>
                            </div>
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}

            <!-- Suspicious patterns -->
            {{if .Report.TimingAnalysis.SuspiciousPatterns}}
            <div>
                <h4 class="font-semibold text-gray-700 mb-3">⚠️ Suspicious Patterns</h4>
                <div class="space-y-2">
                    {{range .Report.TimingAnalysis.SuspiciousPatterns}}
                    <div class="bg-yellow-50 border border-yellow-200 rounded-lg p-3">
                        <p class="text-sm text-yellow-800">{{.}}</p>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>
        {{end}}

        <!-- Downscore Indicators -->
        {{if .Report.DownscoreIndicators}}
        <div class="bg-red-50 border border-red-200 rounded-lg shadow-lg p-6 mb-8">
            <h3 class="text-xl font-bold text-red-800 mb-4">🚨 Peer Scoring Issues Detected</h3>
            <div class="space-y-3">
                {{range .Report.DownscoreIndicators}}
                <div class="bg-white border border-red-300 rounded-lg p-4">
                    <div class="flex items-start">
                        <div class="flex-shrink-0">
                            <svg class="h-5 w-5 text-red-400 mt-0.5" fill="none" viewBox="0 0 20 20" stroke="currentColor">
                                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.664-.833-2.464 0L4.35 16.5c-.77.833.192 2.5 1.732 2.5z"/>
                            </svg>
                        </div>
                        <div class="ml-3">
                            <p class="text-sm text-red-700">{{.}}</p>
                        </div>
                    </div>
                </div>
                {{end}}
            </div>
        </div>
        {{end}}

        <!-- Peer Details -->
        <div class="bg-white rounded-lg shadow-lg p-6 mb-8">
            <h3 class="text-xl font-bold mb-4">Peer Details</h3>
            <p class="text-sm text-gray-600 mb-4">Click on any row to see detailed timing information</p>

            <div class="overflow-x-auto">
                <table id="peer-details-table" class="min-w-full bg-white border border-gray-200 rounded-lg">
                    <thead class="bg-gray-50">
                        <tr>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors" onclick="sortTable(0, 'string')" title="Click to sort">
                                <div class="flex items-center justify-between">
                                    <span>Peer ID</span>
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4"></path>
                                    </svg>
                                </div>
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors" onclick="sortTable(1, 'string')" title="Click to sort">
                                <div class="flex items-center justify-between">
                                    <span>Client</span>
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4"></path>
                                    </svg>
                                </div>
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors" onclick="sortTable(2, 'string')" title="Click to sort">
                                <div class="flex items-center justify-between">
                                    <span>Status</span>
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4"></path>
                                    </svg>
                                </div>
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors" onclick="sortTable(3, 'duration')" title="Click to sort">
                                <div class="flex items-center justify-between">
                                    <span>Duration</span>
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4"></path>
                                    </svg>
                                </div>
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors" onclick="sortTable(4, 'number')" title="Click to sort">
                                <div class="flex items-center justify-between">
                                    <span>Goodbyes</span>
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4"></path>
                                    </svg>
                                </div>
                            </th>
                            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors" onclick="sortTable(5, 'string')" title="Click to sort">
                                <div class="flex items-center justify-between">
                                    <span>Primary Issue</span>
                                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4"></path>
                                    </svg>
                                </div>
                            </th>
                        </tr>
                    </thead>
                    <tbody class="bg-white divide-y divide-gray-200">
                        {{range $peerID, $peer := .Report.Peers}}
                        <tr class="hover:bg-gray-50 cursor-pointer transition-colors duration-150 {{if $peer.LastGoodbye}}{{if or (eq $peer.LastGoodbye "peer score too low") (eq $peer.LastGoodbye "client banned this node") (eq $peer.LastGoodbye "irrelevant network") (eq $peer.LastGoodbye "unable to verify network") (eq $peer.LastGoodbye "fault/error")}}bg-red-50 hover:bg-red-100{{else if or (eq $peer.LastGoodbye "client has too many peers") (eq $peer.LastGoodbye "client shutdown")}}bg-blue-50 hover:bg-blue-100{{else}}bg-yellow-50 hover:bg-yellow-100{{end}}{{end}}"
                             onclick="togglePeerDetails('{{$peerID}}')">
                            <td class="px-6 py-4 whitespace-nowrap">
                                <div class="flex items-center">
                                    <div class="text-sm font-mono text-gray-900">{{$peerID | slice 0 12}}...</div>
                                    <svg class="ml-2 h-4 w-4 text-gray-400 transform transition-transform duration-200" id="arrow-{{$peerID}}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/>
                                    </svg>
                                </div>
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap">
                                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800 capitalize">
                                    {{$peer.ClientType}}
                                </span>
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap">
                                {{if $peer.Disconnected}}
                                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                                        Disconnected
                                    </span>
                                {{else}}
                                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                                        Connected
                                    </span>
                                {{end}}
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                                {{$peer.ConnectionDuration.Round (time.Second)}}
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                                {{if eq $peer.GoodbyeCount 0}}
                                    <span class="text-green-600">None</span>
                                {{else}}
                                    {{$peer.GoodbyeCount}}
                                {{end}}
                            </td>
                            <td class="px-6 py-4 whitespace-nowrap">
                                {{if $peer.LastGoodbye}}
                                    <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {{if or (eq $peer.LastGoodbye "peer score too low") (eq $peer.LastGoodbye "client banned this node") (eq $peer.LastGoodbye "irrelevant network") (eq $peer.LastGoodbye "unable to verify network") (eq $peer.LastGoodbye "fault/error")}}bg-red-100 text-red-800{{else if or (eq $peer.LastGoodbye "client has too many peers") (eq $peer.LastGoodbye "client shutdown")}}bg-blue-100 text-blue-800{{else}}bg-yellow-100 text-yellow-800{{end}}">
                                        {{if gt (len $peer.LastGoodbye) 20}}{{$peer.LastGoodbye | slice 0 18}}...{{else}}{{$peer.LastGoodbye}}{{end}}
                                    </span>
                                {{else}}
                                    <span class="text-green-600">None</span>
                                {{end}}
                            </td>
                        </tr>
                        <tr id="details-{{$peerID}}" class="hidden">
                            <td colspan="6" class="px-6 py-4 bg-gray-50">
                                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                                    <div>
                                        <h4 class="font-medium text-gray-700 mb-2">Connection Details</h4>
                                        <div class="space-y-1 text-sm">
                                            <div><span class="text-gray-600">Connected at:</span> {{$peer.ConnectedAt.Format "15:04:05.000"}}</div>
                                            {{if $peer.Disconnected}}
                                            <div><span class="text-gray-600">Disconnected at:</span> {{$peer.DisconnectedAt.Format "15:04:05.000"}}</div>
                                            {{end}}
                                            <div><span class="text-gray-600">Handshake:</span>
                                                {{if $peer.HandshakeOK}}
                                                    <span class="text-green-600">✓ Success</span>
                                                {{else}}
                                                    <span class="text-red-600">✗ Failed</span>
                                                {{end}}
                                            </div>
                                            <div><span class="text-gray-600">Reconnections:</span> {{$peer.ReconnectionAttempts}}</div>
                                        </div>
                                    </div>

                                    <div>
                                        <h4 class="font-medium text-gray-700 mb-2">Timing Analysis</h4>
                                        <div class="space-y-1 text-sm">
                                            {{if not $peer.FirstGoodbyeAt.IsZero}}
                                            <div><span class="text-gray-600">First goodbye at:</span> {{$peer.FirstGoodbyeAt.Format "15:04:05.000"}}</div>
                                            <div><span class="text-gray-600">Time to first goodbye:</span> {{$peer.TimeToFirstGoodbye.Round (time.Second)}}</div>
                                            {{else}}
                                            <div><span class="text-gray-600">First goodbye:</span> <span class="text-green-600">Never</span></div>
                                            {{end}}
                                            <div><span class="text-gray-600">Total duration:</span> {{$peer.ConnectionDuration.Round (time.Second)}}</div>
                                        </div>
                                    </div>

                                    {{if gt (len $peer.GoodbyeTimings) 0}}
                                    <div>
                                        <h4 class="font-medium text-gray-700 mb-2">Goodbye History</h4>
                                        <div class="space-y-1 text-sm max-h-24 overflow-y-auto">
                                            {{range $peer.GoodbyeTimings}}
                                            <div class="flex justify-between">
                                                <span class="text-gray-600">{{.Timestamp.Format "15:04:05"}}</span>
                                                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium {{if or (eq .Reason "peer score too low") (eq .Reason "client banned this node") (eq .Reason "irrelevant network") (eq .Reason "unable to verify network") (eq .Reason "fault/error")}}bg-red-100 text-red-700{{else if or (eq .Reason "client has too many peers") (eq .Reason "client shutdown")}}bg-blue-100 text-blue-700{{else}}bg-yellow-100 text-yellow-700{{end}}">
                                                    {{.Reason}}
                                                </span>
                                            </div>
                                            {{end}}
                                        </div>
                                    </div>
                                    {{end}}
                                </div>
                            </td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>

            <div class="mt-4 text-sm text-gray-600">
                <p>
                    <span class="inline-block w-3 h-3 bg-red-100 rounded mr-1"></span> Error level (peer score too low, banned, network issues)
                    <span class="inline-block w-3 h-3 bg-blue-100 rounded mr-1 ml-3"></span> Normal level (too many peers, client shutdown)
                    <span class="inline-block w-3 h-3 bg-yellow-100 rounded mr-1 ml-3"></span> Unknown reasons
                </p>
            </div>
        </div>

        <script>
        function togglePeerDetails(peerId) {
            const detailsRow = document.getElementById('details-' + peerId);
            const arrow = document.getElementById('arrow-' + peerId);

            if (detailsRow.classList.contains('hidden')) {
                detailsRow.classList.remove('hidden');
                arrow.style.transform = 'rotate(90deg)';
            } else {
                detailsRow.classList.add('hidden');
                arrow.style.transform = 'rotate(0deg)';
            }
        }

        let lastSortColumn = -1;
        let sortAscending = true;

        function sortTable(columnIndex, dataType) {
            // Find the peer details table specifically by ID
            const table = document.querySelector('#peer-details-table tbody');
            const rows = Array.from(table.querySelectorAll('tr')).filter(row => !row.id.startsWith('details-'));

            // Store details rows before clearing the table
            const detailsRows = new Map();
            rows.forEach(row => {
                const onclickAttr = row.getAttribute('onclick');
                if (onclickAttr) {
                    const peerIdMatch = onclickAttr.match(/togglePeerDetails\('([^']+)'\)/);
                    if (peerIdMatch) {
                        const peerId = peerIdMatch[1];
                        const detailsRow = document.getElementById('details-' + peerId);
                        if (detailsRow) {
                            detailsRows.set(peerId, detailsRow.cloneNode(true));
                        }
                    }
                }
            });

            // Toggle sort direction if same column clicked
            if (lastSortColumn === columnIndex) {
                sortAscending = !sortAscending;
            } else {
                sortAscending = true;
                lastSortColumn = columnIndex;
            }

            rows.sort((a, b) => {
                let aValue = a.cells[columnIndex].textContent.trim();
                let bValue = b.cells[columnIndex].textContent.trim();

                // Handle different data types
                if (dataType === 'number') {
                    aValue = parseInt(aValue) || 0;
                    bValue = parseInt(bValue) || 0;
                } else if (dataType === 'duration') {
                    aValue = parseDuration(aValue);
                    bValue = parseDuration(bValue);
                } else {
                    // String comparison (case insensitive)
                    aValue = aValue.toLowerCase();
                    bValue = bValue.toLowerCase();
                }

                if (aValue < bValue) return sortAscending ? -1 : 1;
                if (aValue > bValue) return sortAscending ? 1 : -1;
                return 0;
            });

            // Clear the table and re-add sorted rows with their detail rows
            table.innerHTML = '';
            rows.forEach(row => {
                table.appendChild(row);
                // Also append the details row if it exists
                const onclickAttr = row.getAttribute('onclick');
                if (onclickAttr) {
                    const peerIdMatch = onclickAttr.match(/togglePeerDetails\('([^']+)'\)/);
                    if (peerIdMatch) {
                        const peerId = peerIdMatch[1];
                        const detailsRow = detailsRows.get(peerId);
                        if (detailsRow) {
                            table.appendChild(detailsRow);
                        }
                    }
                }
            });
        }

        function parseDuration(durationStr) {
            // Parse duration strings like "1m18s", "45s", "2h30m"
            if (durationStr === 'Still connected' || durationStr === '-') return 0;

            let totalSeconds = 0;
            const parts = durationStr.match(/(\d+)([hms])/g);

            if (parts) {
                parts.forEach(part => {
                    const value = parseInt(part.slice(0, -1));
                    const unit = part.slice(-1);

                    switch(unit) {
                        case 'h': totalSeconds += value * 3600; break;
                        case 'm': totalSeconds += value * 60; break;
                        case 's': totalSeconds += value; break;
                    }
                });
            }

            return totalSeconds;
        }
        </script>

        <!-- Test Details -->
        <div class="bg-white rounded-lg shadow-lg p-6">
            <h3 class="text-xl font-bold mb-4">Test Details</h3>
            <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                    <h4 class="font-semibold text-gray-700 mb-2">Test Configuration</h4>
                    <div class="space-y-2 text-sm">
                        <div class="flex justify-between">
                            <span class="text-gray-600">Start Time:</span>
                            <span>{{.Report.StartTime.Format "2006-01-02 15:04:05 UTC"}}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-600">End Time:</span>
                            <span>{{.Report.EndTime.Format "2006-01-02 15:04:05 UTC"}}</span>
                        </div>
                    </div>
                </div>
                <div>
                    <h4 class="font-semibold text-gray-700 mb-2">Results Summary</h4>
                    <div class="space-y-2 text-sm">
                        <div class="flex justify-between">
                            <span class="text-gray-600">Failed Handshakes:</span>
                            <span>{{.Report.FailedHandshakes}}</span>
                        </div>
                        <div class="flex justify-between">
                            <span class="text-gray-600">Total Peers:</span>
                            <span>{{len .Report.Peers}}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Footer -->
        <div class="text-center mt-8 py-4 text-gray-500 text-sm">
            <p>Generated by Hermes Peer Score Tool | <a href="https://github.com/ethpandaops/hermes-peer-score" class="text-blue-600 hover:underline">View Source</a></p>
        </div>
    </div>
</body>
</html>`

// GenerateHTMLReport creates an HTML report from a JSON report file.
// It reads the JSON data, processes it for HTML presentation, and generates
// a styled web page with comprehensive peer connectivity analysis.
//
// Parameters:
//   - jsonFile: Path to the input JSON report file
//   - outputFile: Path where the HTML report should be written
//
// Returns an error if file operations or template processing fails.
func GenerateHTMLReport(jsonFile, outputFile string) error {
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

	// Prepare template data with enhanced fields for HTML presentation.
	// This includes computed fields not present in the raw JSON report.
	templateData := HTMLTemplateData{
		GeneratedAt: time.Now(),
		Report:      report,
	}

	// Calculate connection success rate as a percentage for display.
	if report.TotalConnections > 0 {
		templateData.ConnectionRate = float64(report.SuccessfulHandshakes) / float64(report.TotalConnections) * 100
	}

	// Classify the overall score into human-readable categories.
	// These classifications help users quickly understand their network health.
	if report.ConnectionFailed {
		templateData.ScoreClassification = "Connection Failed"
	} else if report.OverallScore >= 90 {
		templateData.ScoreClassification = "Excellent" // 90-100%: Outstanding connectivity.
	} else if report.OverallScore >= 80 {
		templateData.ScoreClassification = "Good" // 80-89%: Good network health.
	} else if report.OverallScore >= 60 {
		templateData.ScoreClassification = "Fair" // 60-79%: Acceptable but could improve.
	} else if report.OverallScore >= 40 {
		templateData.ScoreClassification = "Poor" // 40-59%: Concerning network issues.
	} else {
		templateData.ScoreClassification = "Critical" // 0-39%: Severe connectivity problems.
	}

	// Convert the client distribution map to a sorted list for consistent HTML display.
	// This ensures client types are presented in a predictable order in the web interface.
	for client, count := range report.PeersByClient {
		templateData.ClientList = append(templateData.ClientList, ClientStat{
			Name:  client,
			Count: count,
		})
	}

	// Create the HTML template with custom helper functions.
	// These functions provide additional functionality within the template context.
	tmpl := template.New("report").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b }, // Mathematical addition for template calculations.
		"time": func() struct{ Second, Millisecond time.Duration } {
			// Provides access to time units for duration formatting in templates.
			return struct{ Second, Millisecond time.Duration }{Second: time.Second, Millisecond: time.Millisecond}
		},
		"slice": func(start, end int, s string) string {
			// String slicing function for templates
			if start < 0 || start >= len(s) {
				return s
			}
			if end < 0 || end > len(s) {
				end = len(s)
			}
			if start >= end {
				return s
			}
			return s[start:end]
		},
	})

	// Parse the HTML template string into a usable template object.
	tmpl, err = tmpl.Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Ensure the output directory exists before attempting to write the file.
	if mkErr := os.MkdirAll(filepath.Dir(outputFile), 0755); mkErr != nil {
		return fmt.Errorf("failed to create output directory: %w", mkErr)
	}

	// Create the output HTML file for writing.
	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Execute the template with our data to generate the final HTML.
	if err := tmpl.Execute(file, templateData); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	log.Printf("HTML report generated: %s", outputFile)

	return nil
}
