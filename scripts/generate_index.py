#!/usr/bin/env python3
"""
Generate the historical index page and reports manifest.
"""
import os
import json
import re
from datetime import datetime, timedelta
from pathlib import Path


def parse_report_metadata(json_file):
    """Extract metadata from a JSON report file."""
    try:
        with open(json_file, 'r') as f:
            data = json.load(f)

        # Calculate success rate
        total_connections = data.get('total_connections', 0)
        successful_handshakes = data.get('successful_handshakes', 0)
        success_rate = (successful_handshakes / total_connections * 100) if total_connections > 0 else 0

        # Extract validation mode and related metadata
        validation_mode = data.get('validation_mode', 'delegated')
        validation_config = data.get('validation_config', {})

        # Extract basic metadata (removed hardcoded validation metrics)
        hermes_version = validation_config.get('hermes_version', 'unknown')

        # Since we removed hardcoded metrics, we'll use basic connection success as a proxy
        validation_success_rate = success_rate  # Use connection success rate

        # For display purposes, calculate simple metrics from actual data
        peers_data = data.get('peers', {})
        total_messages = sum(peer.get('total_message_count', 0) for peer in peers_data.values())
        avg_latency_ms = 0  # No longer tracked in hardcoded metrics
        messages_per_sec = round(total_messages / (data.get('duration', 1) / 1000000000), 1) if data.get('duration') else 0
        error_rate = round(100 - success_rate, 2) if success_rate < 100 else 0  # Inverse of success rate

        # Resource metrics are no longer hardcoded
        cpu_usage = 0
        memory_usage = 0
        cache_hit_rate = 0

        return {
            'unique_peers': len(data.get('peers', {})),
            'total_connections': total_connections,
            'successful_handshakes': successful_handshakes,
            'success_rate': round(success_rate, 1),
            'test_duration': round(data.get('duration', 0) / 1000000000, 1) if data.get('duration') else 0,
            'has_ai_analysis': bool(data.get('ai_analysis')),
            'validation_mode': validation_mode,
            'hermes_version': hermes_version,
            'validation_success_rate': round(validation_success_rate * 100, 1),
            'avg_latency_ms': avg_latency_ms,
            'messages_per_sec': messages_per_sec,
            'error_rate': error_rate,
            'cpu_usage': cpu_usage,
            'memory_usage': memory_usage,
            'cache_hit_rate': cache_hit_rate
        }
    except Exception as e:
        print(f"Error parsing {json_file}: {e}")
        return None


def generate_reports_grid_html(reports):
    """Generate the HTML for the reports grid with validation mode support."""
    reports_grid_html = '<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6" id="reportsGrid">'

    for report in reports:
        html_path = report.get('html_path', '#')
        html_link = f'href="{html_path}"' if html_path and html_path != '#' else 'href="#" onclick="alert(\'HTML report not available for this date\')"'

        # Validation mode styling
        validation_mode = report.get('validation_mode', 'delegated')
        mode_class = f'validation-mode-{validation_mode}'

        # Mode-specific styling and icons
        if validation_mode == 'independent':
            mode_color = 'text-green-600'
            mode_bg = 'bg-green-50'
            mode_border = 'border-green-200'
            mode_icon = '⚡'
            mode_label = 'Independent'
        else:
            mode_color = 'text-blue-600'
            mode_bg = 'bg-blue-50'
            mode_border = 'border-blue-200'
            mode_icon = '🔗'
            mode_label = 'Delegated'

        # Extract basic info
        hermes_version = report.get('hermes_version', 'unknown')

        reports_grid_html += f'''
            <div class="report-card bg-white rounded-lg shadow-md p-6 data-report {mode_class}"
                 data-date="{report['date']}"
                 data-peers="{report['unique_peers']}"
                 data-connections="{report['total_connections']}"
                 data-success="{report['success_rate']}"
                 data-validation-mode="{validation_mode}"
                 data-hermes-version="{hermes_version}">
                <div class="flex items-center justify-between mb-4">
                    <div>
                        <h3 class="text-lg font-semibold text-gray-900">{report['formatted_date']}</h3>
                        <div class="text-xs text-gray-500 mt-1">{hermes_version}</div>
                        <div class="mt-1">
                            <span class="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium {mode_bg} {mode_color} {mode_border} border">
                                {mode_icon} {mode_label}
                            </span>
                        </div>
                    </div>
                    <div class="text-right">
                        <div class="text-sm font-medium {mode_color}">{report['test_duration']}s</div>
                    </div>
                </div>

                <div class="flex space-x-2">
                    <a {html_link}
                       class="flex-1 inline-flex items-center justify-center px-3 py-2 bg-gradient-to-r from-blue-600 to-indigo-600 text-white rounded text-sm font-medium hover:from-blue-700 hover:to-indigo-700 transition-all">
                        📊 View Report
                    </a>
                    <a href="{report['json_path']}"
                       class="inline-flex items-center justify-center px-3 py-2 bg-gray-600 text-white rounded text-sm font-medium hover:bg-gray-700 transition-colors">
                        📄 JSON
                    </a>
                </div>
            </div>'''

    reports_grid_html += '</div>'
    return reports_grid_html


def generate_latest_report_html(latest_report):
    """Generate HTML for the latest report section with validation mode support."""
    validation_mode = latest_report.get('validation_mode', 'delegated')
    hermes_version = latest_report.get('hermes_version', 'unknown')

    # Mode-specific styling and content
    if validation_mode == 'independent':
        bg_gradient = 'from-green-50 to-emerald-50'
        border_color = 'border-green-200'
        title_color = 'text-green-900'
        text_color = 'text-green-700'
        accent_color = 'text-green-600'
        value_color = 'text-green-800'
        button_color = 'bg-green-600 hover:bg-green-700'
        mode_icon = '⚡'
        mode_title = 'Independent Validation'
    else:
        bg_gradient = 'from-blue-50 to-indigo-50'
        border_color = 'border-blue-200'
        title_color = 'text-blue-900'
        text_color = 'text-blue-700'
        accent_color = 'text-blue-600'
        value_color = 'text-blue-800'
        button_color = 'bg-blue-600 hover:bg-blue-700'
        mode_icon = '🔗'
        mode_title = 'Delegated Validation'

    # Get basic info
    hermes_version = latest_report.get('hermes_version', 'unknown')

    return f'''<div class="bg-gradient-to-r {bg_gradient} border {border_color} rounded-lg shadow p-6 mb-6">
        <div class="flex items-center justify-between">
            <div>
                <div class="flex items-center space-x-2 mb-2">
                    <h2 class="text-xl font-semibold {title_color}">Latest Report</h2>
                    <span class="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-white {accent_color} border {border_color}">
                        {mode_icon} {mode_title}
                    </span>
                </div>
                <p class="{text_color} mb-3">{latest_report['formatted_date']}</p>
                <div class="grid grid-cols-2 md:grid-cols-2 gap-4 text-sm">
                    <div>
                        <span class="{accent_color} font-medium">Duration:</span>
                        <span class="{value_color}">{latest_report['test_duration']}s</span>
                    </div>
                    <div>
                        <span class="{accent_color} font-medium">Version:</span>
                        <span class="{value_color}">{hermes_version}</span>
                    </div>
                </div>
            </div>
            <div class="flex space-x-3">
                <a href="{latest_report['html_path'] or '#'}"
                   class="inline-flex items-center px-4 py-2 {button_color} text-white rounded-md text-sm font-medium transition-colors">
                    📊 View Report
                </a>
                <a href="{latest_report['json_path']}"
                   class="inline-flex items-center px-4 py-2 bg-gray-600 text-white rounded-md text-sm font-medium hover:bg-gray-700 transition-colors">
                    📄 Raw Data
                </a>
            </div>
        </div>
    </div>'''


def clean_html_template_syntax(html):
    """Clean up any remaining template syntax - conservative approach."""
    # Remove complete template blocks that weren't handled
    html = re.sub(r'\{\{if [^}]+\}\}.*?\{\{end\}\}', '', html, flags=re.DOTALL)
    html = re.sub(r'\{\{range [^}]+\}\}.*?\{\{end\}\}', '', html, flags=re.DOTALL)

    # Remove any standalone {{.Variable}} that weren't replaced
    html = re.sub(r'\{\{[^}]+\}\}', '', html)

    # Fix href attributes
    html = re.sub(r'href=""[^>]*class=', 'href="#" class=', html)

    # Remove simple template fragments
    html = re.sub(r'^\s*">[^<]*$', '', html, flags=re.MULTILINE)
    html = re.sub(r'^\s*/$', '', html, flags=re.MULTILINE)

    # Remove double empty lines
    html = re.sub(r'\n\s*\n\s*\n', '\n\n', html)

    return html


def generate_reports_manifest(reports):
    """Generate the reports manifest JSON file."""
    manifest_data = {
        "generated_at": datetime.utcnow().isoformat() + "Z",
        "total_reports": len(reports),
        "reports": []
    }

    for report in reports:
        report_files = []

        # Add JSON file
        json_filename = f"peer-score-report-{report['timestamp']}.json"
        report_files.append({
            "filename": json_filename,
            "path": f"{report['date']}/{json_filename}",
            "type": "json"
        })

        # Add HTML file if it exists
        if report.get('html_path'):
            html_filename = f"peer-score-report-{report['timestamp']}.html"
            report_files.append({
                "filename": html_filename,
                "path": f"{report['date']}/{html_filename}",
                "type": "html"
            })

        # Add JS data file
        js_filename = f"peer-score-report-{report['timestamp']}-data.js"
        report_files.append({
            "filename": js_filename,
            "path": f"{report['date']}/{js_filename}",
            "type": "javascript"
        })

        manifest_data["reports"].append({
            "date": report['date'],
            "timestamp": report['timestamp'],
            "formatted_date": report['formatted_date'],
            "test_duration": report['test_duration'],
            "unique_peers": report['unique_peers'],
            "total_connections": report['total_connections'],
            "success_rate": report['success_rate'],
            "validation_mode": report.get('validation_mode', 'delegated'),
            "hermes_version": report.get('hermes_version', 'unknown'),
            "messages_per_sec": report.get('messages_per_sec', 0),
            "files": report_files
        })

    return manifest_data


def generate_index():
    """Generate the index.html file with all historical reports (28-day retention)."""
    reports_dir = Path('reports')
    reports = []

    # Calculate cutoff date (28 days ago)
    cutoff_date = datetime.now() - timedelta(days=28)

    # Find all JSON reports (within 28-day retention period)
    for json_file in reports_dir.glob('**/peer-score-report-*.json'):
        date_dir = json_file.parent.name
        filename = json_file.name

        # Extract timestamp from filename - handle both old and new formats
        timestamp_part = filename.replace('peer-score-report-', '').replace('.json', '')

        # Handle new format: delegated-YYYY-MM-DD_HH-MM-SS or independent-YYYY-MM-DD_HH-MM-SS
        if timestamp_part.startswith('delegated-') or timestamp_part.startswith('independent-'):
            # New format: remove validation mode prefix
            if timestamp_part.startswith('delegated-'):
                timestamp_part = timestamp_part[len('delegated-'):]
            elif timestamp_part.startswith('independent-'):
                timestamp_part = timestamp_part[len('independent-'):]
        else:
            # Old format: remove validation mode suffixes if present
            for suffix in ['-delegated', '-independent']:
                if timestamp_part.endswith(suffix):
                    timestamp_part = timestamp_part[:-len(suffix)]
                    break

        try:
            # Parse the timestamp
            report_date = datetime.strptime(timestamp_part, '%Y-%m-%d_%H-%M-%S')

            # Skip reports older than 28 days
            if report_date < cutoff_date:
                print(f"Skipping old report: {timestamp_part}")
                continue

            # Get metadata
            metadata = parse_report_metadata(json_file)
            if metadata is None:
                continue

            # Determine file paths
            html_file = json_file.with_suffix('.html')

            html_path = f"{date_dir}/{html_file.name}" if html_file.exists() else None
            json_path = f"{date_dir}/{json_file.name}"

            # For display purposes, use the original timestamp + validation mode
            display_timestamp = filename.replace('peer-score-report-', '').replace('.json', '')

            reports.append({
                'date': report_date.strftime('%Y-%m-%d'),
                'timestamp': display_timestamp,  # Use full timestamp with validation mode
                'formatted_date': report_date.strftime('%B %d, %Y at %H:%M'),
                'html_path': html_path,
                'json_path': json_path,
                **metadata
            })
        except ValueError as e:
            print(f"Could not parse timestamp from {filename}: {e}")
            continue

    # Sort by timestamp (newest first) - this ensures proper ordering when multiple reports exist for the same day
    reports.sort(key=lambda x: x['timestamp'], reverse=True)

    # Prepare template data
    template_data = {
        'total_reports': len(reports),
        'latest_report': reports[0] if reports else None,
        'reports': reports,
        'last_updated': datetime.utcnow().strftime('%B %d, %Y at %H:%M UTC')
    }

    # Read template
    with open('index-template.html', 'r') as f:
        template = f.read()

    # Simple template replacement
    html = template.replace('{{.TotalReports}}', str(template_data['total_reports']))
    html = html.replace('{{.LastUpdated}}', template_data['last_updated'])

    # Handle latest report section
    if template_data['latest_report']:
        latest_block = generate_latest_report_html(template_data['latest_report'])
        html = re.sub(r'\{\{if \.LatestReport\}\}.*?\{\{end\}\}', latest_block, html, flags=re.DOTALL)
    else:
        # Remove the latest report section entirely
        html = re.sub(r'\{\{if \.LatestReport\}\}.*?\{\{end\}\}', '', html, flags=re.DOTALL)

    # Generate and replace reports grid - replace the entire template section
    reports_grid_html = generate_reports_grid_html(reports)

    # Replace the entire grid section including the template range using regex
    grid_pattern = r'<!-- Reports Grid -->\s*<div class="grid[^>]*?id="reportsGrid"[^>]*>\s*\{\{range [^}]+\}\}.*?\{\{end\}\}\s*</div>'
    grid_replacement = f'<!-- Reports Grid -->\n        {reports_grid_html}'
    html = re.sub(grid_pattern, grid_replacement, html, flags=re.DOTALL)

    # Clean up any remaining template syntax
    html = clean_html_template_syntax(html)

    # Write the final index
    with open('reports/index.html', 'w') as f:
        f.write(html)

    # Generate and write manifest
    manifest_data = generate_reports_manifest(reports)
    with open('reports/reports-manifest.json', 'w') as f:
        json.dump(manifest_data, f, indent=2)

    print(f"Generated index.html with {len(reports)} reports")
    print(f"Generated reports-manifest.json with {len(reports)} report entries")
    return len(reports)


if __name__ == '__main__':
    generate_index()
