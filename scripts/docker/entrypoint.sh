#!/bin/sh
if [ "$KYMA_FIPS_MODE_ENABLED" = "true" ]; then
  export GODEBUG="fips140=only"
fi

echo "FIPS mode enabled: ${KYMA_FIPS_MODE_ENABLED:-false}"

# Run the original binary
exec "$@"