#!/bin/bash

for chart in charts/*/; do
if [ -d "${chart}" ]; then
    chart_name=$(basename "${chart}")
    
    # Update dependencies
    helm dependency update "${chart}"
    
    # Package chart
    helm package "${chart}"
    
    # Push to OCI registry
    helm push "${chart_name}"-*.tgz oci://ghcr.io/bartlettc22/charts
    
    # Clean up
	rm "${chart_name}"-*.tgz
fi
done