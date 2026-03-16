#!/bin/bash
# Cleanup Python cache and bytecode files

echo "Cleaning up Python cache files..."

# Remove __pycache__ directories
find tests -type d -name "__pycache__" -exec rm -rf {} + 2>/dev/null || true

# Remove .pyc files
find tests -type f -name "*.pyc" -delete 2>/dev/null || true

# Remove .pyo files
find tests -type f -name "*.pyo" -delete 2>/dev/null || true

echo "Cleanup complete!"
