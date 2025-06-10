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

        return {
            'unique_peers': len(data.get('peers', {})),
            'total_connections': total_connections,
            'successful_handshakes': successful_handshakes,
            'success_rate': round(success_rate, 1),
            'test_duration': round(data.get('duration', 0) / 1000000000, 1) if data.get('duration') else 0,
            'has_ai_analysis': bool(data.get('ai_analysis'))
        }
    except Exception as e:
        print(f"Error parsing {json_file}: {e}")
        return None


def generate_reports_grid_html(reports):
    """Generate the HTML for the reports grid (simplified version)."""
    reports_grid_html = '<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6" id="reportsGrid">'
    
    for report in reports:
        html_path = report.get('html_path', '#')
        html_link = f'href="{html_path}"' if html_path and html_path != '#' else 'href="#" onclick="alert(\'HTML report not available for this date\')"'

        reports_grid_html += f'''
            <div class="report-card bg-white rounded-lg shadow-md p-6 data-report"
                 data-date="{report['date']}"
                 data-peers="{report['unique_peers']}"
                 data-connections="{report['total_connections']}"
                 data-success="{report['success_rate']}">
                <div class="flex items-center justify-between mb-4">
                    <div>
                        <h3 class="text-lg font-semibold text-gray-900">{report['formatted_date']}</h3>
                        <p class="text-sm text-gray-600">{report['date']}</p>
                    </div>
                    <div class="text-right">
                        <div class="text-xs text-gray-500">Duration</div>
                        <div class="text-sm font-medium text-blue-600">{report['test_duration']}s</div>
                    </div>
                </div>

                <div class="flex space-x-2">
                    <a {html_link}
                       class="flex-1 inline-flex items-center justify-center px-3 py-2 bg-blue-600 text-white rounded text-sm font-medium hover:bg-blue-700 transition-colors">
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
    """Generate HTML for the latest report section."""
    return f'''<div class="bg-gradient-to-r from-blue-50 to-indigo-50 border border-blue-200 rounded-lg shadow p-6 mb-6">
        <div class="flex items-center justify-between">
            <div>
                <div class="flex items-center space-x-2 mb-2">
                    <h2 class="text-xl font-semibold text-blue-900">Latest Report</h2>
                </div>
                <p class="text-blue-700 mb-3">{latest_report['date']} - {latest_report['formatted_date']}</p>
                <div class="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                    <div>
                        <span class="text-blue-600 font-medium">Duration:</span>
                        <span class="text-blue-800">{latest_report['test_duration']}s</span>
                    </div>
                    <div>
                        <span class="text-blue-600 font-medium">Peers:</span>
                        <span class="text-blue-800">{latest_report['unique_peers']}</span>
                    </div>
                    <div>
                        <span class="text-blue-600 font-medium">Connections:</span>
                        <span class="text-blue-800">{latest_report['total_connections']}</span>
                    </div>
                    <div>
                        <span class="text-blue-600 font-medium">Success Rate:</span>
                        <span class="text-blue-800">{latest_report['success_rate']}%</span>
                    </div>
                </div>
            </div>
            <div class="flex space-x-3">
                <a href="{latest_report['html_path'] or '#'}"
                   class="inline-flex items-center px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 transition-colors">
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
    """Clean up any remaining template syntax."""
    # Remove complete template blocks that weren't handled
    html = re.sub(r'\{\{if [^}]+\}\}.*?\{\{end\}\}', '', html, flags=re.DOTALL)
    
    # Remove any standalone {{.Variable}} that weren't replaced
    html = re.sub(r'\{\{[^}]+\}\}', '', html)
    
    # Clean up malformed remnants
    html = re.sub(r'">[^<]*%[^<]*</div>', '"></div>', html)
    html = re.sub(r'>\s*">[^<]*</div>', '></div>', html)
    html = re.sub(r'href=""[^>]*class=', 'href="#" class=', html)
    
    # Remove any lines that are just template fragments
    lines = html.split('\n')
    cleaned_lines = []
    for line in lines:
        # Skip lines that are clearly broken template remnants
        if re.search(r'^\s*">[^<]*%|^\s*</div>\s*">[^<]*%|^\s*/[^<]*</div>', line):
            continue
        if line.strip() == '/' or line.strip().startswith('">'):
            continue
        cleaned_lines.append(line)
    
    return '\n'.join(cleaned_lines)


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

        # Extract timestamp from filename
        timestamp_part = filename.replace('peer-score-report-', '').replace('.json', '')

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

            reports.append({
                'date': report_date.strftime('%Y-%m-%d'),
                'timestamp': timestamp_part,
                'formatted_date': report_date.strftime('%B %d, %Y at %H:%M'),
                'html_path': html_path,
                'json_path': json_path,
                **metadata
            })
        except ValueError as e:
            print(f"Could not parse timestamp from {filename}: {e}")
            continue

    # Sort by date (newest first)
    reports.sort(key=lambda x: x['date'], reverse=True)

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

    # Generate and replace reports grid
    reports_grid_html = generate_reports_grid_html(reports)
    grid_pattern = r'<!-- Reports Grid -->.*?<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6" id="reportsGrid">.*?</div>'
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