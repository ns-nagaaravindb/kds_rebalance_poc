#!/bin/bash

echo "🛑 Stopping Kinesis POC..."
echo ""

# Stop and remove containers
docker-compose down

echo ""
echo "✅ All services stopped!"
echo ""
echo "To remove all data volumes as well, run:"
echo "   docker-compose down -v"
