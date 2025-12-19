# OSINTMCP Deployment Memory

## Successful Cloud Run Deployment Process (2025-10-20)

### Key Learnings from Forecast Chart Deployment

#### Version Compatibility is Critical
1. **Go Version Synchronization**
   - `go.mod` Go version MUST match Dockerfile `FROM golang:X.XX-alpine`
   - Dependencies like `golang.org/x/crypto@v0.43.0` and `golang.org/x/sys@v0.37.0` require Go 1.24+
   - Solution: Use `golang:1.24-alpine` in Dockerfile and `go 1.24` in go.mod

2. **Node Version Requirements**
   - Frontend dependencies (`vite@7.1.9`, `react-router@7.9.4`, `@vitejs/plugin-react@5.0.4`) require Node 20+
   - Solution: Use `node:20-alpine` instead of `node:18-alpine`

3. **TypeScript Strict Mode**
   - Cloud Build fails on unused imports that local dev might ignore
   - Always run `npm run build` locally before deploying
   - Remove all unused imports and variables

#### Successful Deployment Command
```bash
# Get commit SHA
COMMIT_SHA=$(git rev-parse HEAD)
SHORT_SHA=$(git rev-parse --short HEAD)

# Deploy with explicit substitutions
gcloud builds submit --config cloudbuild.yaml \
  --substitutions=COMMIT_SHA=$COMMIT_SHA,SHORT_SHA=$SHORT_SHA
```

#### Common Build Failures & Solutions

**Error: "module X requires go >= 1.24.0"**
- Cause: go.mod version < Dockerfile Go version or dependency requirements
- Solution: Upgrade both go.mod and Dockerfile to Go 1.24+

**Error: "Unsupported engine" (Node)**
- Cause: Node version in Dockerfile too old for frontend dependencies
- Solution: Upgrade to node:20-alpine or higher

**Error: "TS6133: declared but never used"**
- Cause: Unused imports/variables in TypeScript
- Solution: Remove unused imports before committing

#### Architecture Notes
- Multi-stage build: frontend → backend → runtime
- Frontend built with Node 20, outputs to dist/
- Backend built with Go 1.24, outputs binary
- Runtime uses debian:bookworm-slim for compatibility
- Build time: ~2.5 minutes when successful

#### Pre-Deployment Checklist
1. ✅ Run `npm run build` in /web directory
2. ✅ Check go.mod version matches Dockerfile
3. ✅ Check Node version supports package.json engines
4. ✅ Commit all changes and push to GitHub
5. ✅ Use explicit COMMIT_SHA substitutions in gcloud command
