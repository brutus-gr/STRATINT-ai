#!/bin/bash

# OSINTMCP Development Startup Script

set -e

# Get the directory where the script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Get the project root (parent of scripts directory)
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

echo "🚀 Starting OSINTMCP Development Environment"
echo "=============================================="

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker and try again."
    exit 1
fi

# Trap to cleanup background processes on exit
cleanup() {
    echo ""
    echo -e "${YELLOW}🛑 Shutting down services...${NC}"
    if [ ! -z "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null
    fi
    if [ ! -z "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null
    fi
    docker compose down
    echo -e "${GREEN}✅ All services stopped${NC}"
    exit 0
}

trap cleanup SIGINT SIGTERM

echo -e "${BLUE}📦 Step 1: Starting PostgreSQL with Docker Compose${NC}"
docker compose up -d

echo ""
echo -e "${BLUE}⏳ Step 2: Waiting for PostgreSQL to be healthy...${NC}"
sleep 3

# Wait for PostgreSQL
echo -e "   ${YELLOW}Waiting for PostgreSQL...${NC}"
until docker compose exec -T postgres pg_isready -U stratint > /dev/null 2>&1; do
  echo -e "   ${YELLOW}PostgreSQL is unavailable - waiting...${NC}"
  sleep 2
done
echo -e "   ${GREEN}✓ PostgreSQL is ready${NC}"

echo ""
echo -e "${BLUE}📊 Step 3: Database Status${NC}"
docker compose ps

echo ""
echo -e "${GREEN}✅ Infrastructure is ready!${NC}"

# Load environment variables from .env file
if [ -f .env ]; then
    echo ""
    echo -e "${BLUE}🔧 Loading environment variables from .env${NC}"
    export $(grep -v '^#' .env | xargs)
    echo -e "   ${GREEN}✓ Environment loaded${NC}"
fi

echo ""
echo -e "${BLUE}🚀 Step 4: Starting Go Backend...${NC}"
go run ./cmd/server > /tmp/stratint-backend.log 2>&1 &
BACKEND_PID=$!
echo -e "   ${GREEN}✓ Backend started (PID: $BACKEND_PID)${NC}"
sleep 2

echo ""
echo -e "${BLUE}🎨 Step 5: Starting Frontend...${NC}"
(cd web && npm run dev) > /tmp/stratint-frontend.log 2>&1 &
FRONTEND_PID=$!
echo -e "   ${GREEN}✓ Frontend started (PID: $FRONTEND_PID)${NC}"

echo ""
echo -e "${GREEN}✅ All services are running!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${BLUE}📱 Access the application:${NC}"
echo ""
echo -e "   ${YELLOW}Frontend: http://localhost:5173${NC}"
echo -e "   ${YELLOW}API:      http://localhost:8080${NC}"
echo -e "   ${YELLOW}Admin:    http://localhost:5173/#admin${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${BLUE}📋 Service Status:${NC}"
echo ""
echo -e "   Backend PID:  ${GREEN}$BACKEND_PID${NC}"
echo -e "   Frontend PID: ${GREEN}$FRONTEND_PID${NC}"
echo ""
echo -e "${BLUE}📄 Logs:${NC}"
echo ""
echo -e "   Backend:  tail -f /tmp/stratint-backend.log"
echo -e "   Frontend: tail -f /tmp/stratint-frontend.log"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop all services${NC}"
echo ""

# Wait for processes
wait
