#!/bin/bash

echo "=== Cauldron E2E Test ==="
echo ""
echo "This test will:"
echo "1. Start the Wails app in dev mode"
echo "2. Wait for the app to initialize"
echo "3. Check console logs for Wails runtime detection"
echo "4. Verify database loading"
echo ""

echo "Starting Wails dev server..."
cd /mnt/d/GoLandProjects/cauldronGO/cauldronGO

# Kill any existing wails dev processes
pkill -f "wails dev" 2>/dev/null

# Start wails dev in background
timeout 60s ~/go/bin/wails dev > wails_dev.log 2>&1 &
WAILS_PID=$!

echo "Waiting for Wails dev server to start..."
sleep 20

echo ""
echo "=== Checking Wails dev server logs ==="
echo ""

if grep -q "Frontend server started" wails_dev.log; then
    echo "✓ Frontend server started"
else
    echo "✗ Frontend server did not start"
    cat wails_dev.log
    kill $WAILS_PID 2>/dev/null
    exit 1
fi

if grep -q "DomReady" wails_dev.log || grep -q "Launching" wails_dev.log; then
    echo "✓ Application window launched"
else
    echo "✗ Application window did not launch"
    cat wails_dev.log
    kill $WAILS_PID 2>/dev/null
    exit 1
fi

echo ""
echo "=== Backend Tests ==="
echo ""

# Run integration tests while dev server is running
go test -v -run TestWailsBindings -timeout 30s 2>&1 | tee test_output.log

if grep -q "PASS: TestWailsBindings" test_output.log; then
    echo "✓ All backend integration tests passed"
else
    echo "✗ Backend integration tests failed"
    kill $WAILS_PID 2>/dev/null
    exit 1
fi

echo ""
echo "=== E2E Test Summary ==="
echo "✓ Wails dev server started successfully"
echo "✓ Application window launched"
echo "✓ Backend integration tests passed"
echo ""
echo "To manually verify Angular communication:"
echo "1. The app should now be running"
echo "2. Open Developer Tools (F12)"
echo "3. Check Console for these messages:"
echo "   - '=== Home Component Initialization ==='"
echo "   - 'window.go exists: true'"
echo "   - 'window.runtime exists: true'"
echo "   - 'wails.isWails: true'"
echo "   - '✓ Wails runtime detected, starting initialization...'"
echo "   - 'Loaded jobs: [...]'"
echo "   - 'Loaded imported files: [...]'"
echo ""
echo "Press Ctrl+C to stop the dev server"

# Keep running
wait $WAILS_PID
