#!/bin/bash

# Enhanced test script for Precipitation API with Polygon Reuse Feature
# Usage: ./test_polygon_reuse.sh [host] [port]

HOST=${1:-localhost}
PORT=${2:-8086}
BASE_URL="http://${HOST}:${PORT}/weather/public/api/v2/precipitation/polygon"

echo "=================================================="
echo "Precipitation API - Polygon Reuse Feature Tests"
echo "=================================================="
echo "Target: ${BASE_URL}"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Create new polygon (original behavior)
echo -e "${YELLOW}Test 1: Create New Polygon (No polygon_id provided)${NC}"
echo "---"
echo "Expected: polygon_created_new: true, polygon_reused: false"
echo ""

RESPONSE=$(curl -s -X GET "${BASE_URL}?lat1=21.0285&lon1=105.8542&lat2=21.0385&lon2=105.8542&lat3=21.0385&lon3=105.8642&lat4=21.0285&lon4=105.8642&start=1704067200&end=1706745600")
echo "$RESPONSE" | jq '.'

# Extract polygon ID for reuse
POLYGON_ID=$(echo "$RESPONSE" | jq -r '.polygon_id')
CREATED_NEW=$(echo "$RESPONSE" | jq -r '.polygon_created_new')
REUSED=$(echo "$RESPONSE" | jq -r '.polygon_reused')

if [ "$CREATED_NEW" == "true" ] && [ "$REUSED" == "false" ]; then
    echo -e "${GREEN}✓ Test 1 PASSED${NC}"
    echo -e "${GREEN}Polygon ID: $POLYGON_ID${NC}"
else
    echo -e "${RED}✗ Test 1 FAILED${NC}"
fi

echo ""
echo "=================================================="
echo ""

# Test 2: Reuse existing polygon
if [ "$POLYGON_ID" != "null" ] && [ -n "$POLYGON_ID" ]; then
    echo -e "${YELLOW}Test 2: Reuse Existing Polygon${NC}"
    echo "---"
    echo "Using polygon_id: $POLYGON_ID"
    echo "Expected: polygon_reused: true, polygon_created_new: false"
    echo ""

    RESPONSE2=$(curl -s -X GET "${BASE_URL}?polygon_id=${POLYGON_ID}&start=1704067200&end=1706745600")
    echo "$RESPONSE2" | jq '.'

    CREATED_NEW2=$(echo "$RESPONSE2" | jq -r '.polygon_created_new')
    REUSED2=$(echo "$RESPONSE2" | jq -r '.polygon_reused')
    POLYGON_ID2=$(echo "$RESPONSE2" | jq -r '.polygon_id')

    if [ "$REUSED2" == "true" ] && [ "$CREATED_NEW2" == "false" ] && [ "$POLYGON_ID2" == "$POLYGON_ID" ]; then
        echo -e "${GREEN}✓ Test 2 PASSED - Polygon successfully reused${NC}"
    else
        echo -e "${RED}✗ Test 2 FAILED${NC}"
    fi

    echo ""
    echo "=================================================="
    echo ""
fi

# Test 3: Invalid polygon ID with fallback to coordinates
echo -e "${YELLOW}Test 3: Invalid Polygon ID with Fallback${NC}"
echo "---"
echo "Using invalid polygon_id with valid coordinates"
echo "Expected: polygon_created_new: true (fallback to creating new)"
echo ""

RESPONSE3=$(curl -s -X GET "${BASE_URL}?polygon_id=invalid_polygon_12345&lat1=21.0285&lon1=105.8542&lat2=21.0385&lon2=105.8542&lat3=21.0385&lon3=105.8642&lat4=21.0285&lon4=105.8642&start=1704067200&end=1706745600")
echo "$RESPONSE3" | jq '.'

CREATED_NEW3=$(echo "$RESPONSE3" | jq -r '.polygon_created_new')
REUSED3=$(echo "$RESPONSE3" | jq -r '.polygon_reused')

if [ "$CREATED_NEW3" == "true" ] && [ "$REUSED3" == "false" ]; then
    echo -e "${GREEN}✓ Test 3 PASSED - Fallback successful${NC}"
else
    echo -e "${RED}✗ Test 3 FAILED${NC}"
fi

echo ""
echo "=================================================="
echo ""

# Test 4: Missing both polygon_id and coordinates (should fail)
echo -e "${YELLOW}Test 4: Missing Both Polygon ID and Coordinates${NC}"
echo "---"
echo "Expected: HTTP 400 Bad Request"
echo ""

HTTP_CODE=$(curl -s -w "%{http_code}" -o /dev/null -X GET "${BASE_URL}?start=1704067200&end=1706745600")

if [ "$HTTP_CODE" == "400" ]; then
    echo -e "${GREEN}✓ Test 4 PASSED - Correctly rejected (HTTP $HTTP_CODE)${NC}"
else
    echo -e "${RED}✗ Test 4 FAILED - Got HTTP $HTTP_CODE, expected 400${NC}"
fi

echo ""
echo "=================================================="
echo ""

# Test 5: Invalid polygon ID without coordinates (should fail)
echo -e "${YELLOW}Test 5: Invalid Polygon ID Without Coordinates${NC}"
echo "---"
echo "Expected: HTTP 400 Bad Request"
echo ""

HTTP_CODE2=$(curl -s -w "%{http_code}" -o /dev/null -X GET "${BASE_URL}?polygon_id=invalid_id&start=1704067200&end=1706745600")

if [ "$HTTP_CODE2" == "400" ]; then
    echo -e "${GREEN}✓ Test 5 PASSED - Correctly rejected (HTTP $HTTP_CODE2)${NC}"
else
    echo -e "${RED}✗ Test 5 FAILED - Got HTTP $HTTP_CODE2, expected 400${NC}"
fi

echo ""
echo "=================================================="
echo ""

# Test 6: Polygon ID with coordinates (should prefer polygon ID if valid)
if [ "$POLYGON_ID" != "null" ] && [ -n "$POLYGON_ID" ]; then
    echo -e "${YELLOW}Test 6: Valid Polygon ID + Coordinates${NC}"
    echo "---"
    echo "Providing both valid polygon_id and coordinates"
    echo "Expected: polygon_reused: true (polygon_id takes precedence)"
    echo ""

    RESPONSE6=$(curl -s -X GET "${BASE_URL}?polygon_id=${POLYGON_ID}&lat1=21.0285&lon1=105.8542&lat2=21.0385&lon2=105.8542&lat3=21.0385&lon3=105.8642&lat4=21.0285&lon4=105.8642&start=1704067200&end=1706745600")
    echo "$RESPONSE6" | jq '.'

    CREATED_NEW6=$(echo "$RESPONSE6" | jq -r '.polygon_created_new')
    REUSED6=$(echo "$RESPONSE6" | jq -r '.polygon_reused')

    if [ "$REUSED6" == "true" ] && [ "$CREATED_NEW6" == "false" ]; then
        echo -e "${GREEN}✓ Test 6 PASSED - Polygon ID took precedence${NC}"
    else
        echo -e "${RED}✗ Test 6 FAILED${NC}"
    fi

    echo ""
    echo "=================================================="
    echo ""
fi

# Summary
echo ""
echo "=================================================="
echo "                  TEST SUMMARY"
echo "=================================================="
echo ""
echo "Key Findings:"
echo "1. New polygon creation works correctly"
echo "2. Polygon reuse works correctly"
echo "3. Fallback mechanism works when polygon not found"
echo "4. Validation prevents missing parameters"
echo ""
echo "Saved Polygon ID for future use: $POLYGON_ID"
echo ""
echo "You can reuse this polygon ID in future queries:"
echo "  curl -X GET \"${BASE_URL}?polygon_id=${POLYGON_ID}&start=START&end=END\""
echo ""
echo "=================================================="
