#!/bin/bash
set -e

# This script copies the display.go file from the lib directory to the clock
# directory and updates the import path to match the new location.

cp ../lib/display/display.go ./display/display.go
sed -i'' -e 's|github.com/jaredwarren/clock/lib/config|github.com/jaredwarren/clock/clock/config|' ./display/display.go
