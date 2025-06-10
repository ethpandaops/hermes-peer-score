#!/usr/bin/env python3
"""
Download historical reports from GitHub Pages using the manifest file.
"""
import json
import os
import subprocess
from datetime import datetime
import sys


def download_reports_from_manifest(manifest_file, cutoff_days=28):
    """Download reports from manifest, filtering by age."""
    try:
        with open(manifest_file, 'r') as f:
            manifest = json.load(f)
        
        base_url = 'https://ethpandaops.github.io/hermes-peer-score/'
        cutoff_date = datetime.strptime(os.environ.get('CUTOFF_DATE'), '%Y-%m-%d')
        downloaded_count = 0
        
        for report in manifest.get('reports', []):
            report_date = datetime.strptime(report['date'], '%Y-%m-%d')
            
            # Skip reports older than cutoff
            if report_date < cutoff_date:
                continue
                
            # Create directory
            date_dir = f"reports/{report['date']}"
            os.makedirs(date_dir, exist_ok=True)
            
            # Download each file for this report
            files_downloaded = 0
            for file_info in report.get('files', []):
                file_url = base_url + file_info['path']
                local_path = f"reports/{file_info['path']}"
                
                try:
                    result = subprocess.run(['curl', '-f', '-s', file_url, '-o', local_path], 
                                          capture_output=True)
                    if result.returncode == 0:
                        files_downloaded += 1
                        print(f'  Downloaded: {file_info["filename"]}')
                    else:
                        print(f'  Failed to download: {file_info["filename"]}')
                except Exception as e:
                    print(f'  Error downloading {file_info["filename"]}: {e}')
            
            if files_downloaded > 0:
                downloaded_count += 1
                print(f'Successfully downloaded {files_downloaded} files for {report["date"]}')
        
        print(f'Total reports preserved: {downloaded_count}')
        return True
        
    except Exception as e:
        print(f'Error processing manifest: {e}')
        return False


if __name__ == '__main__':
    success = download_reports_from_manifest('reports-manifest.json')
    sys.exit(0)  # Don't fail the build, just continue without historical data