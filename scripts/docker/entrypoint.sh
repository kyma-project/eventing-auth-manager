#!/bin/sh
if [ "$KYMA_FIPS_MODE_ENABLED" = "true" ]; then
  export GODEBUG="fips140=only"
fi

# Run the original binary
exec "$@"