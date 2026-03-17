#!/bin/bash
# Create a simple test image with EXIF data using ImageMagick if available
# This is a helper script; the actual test uses embedded base64 data as fallback

if command -v convert &> /dev/null; then
    # Create a simple 100x100 red square with EXIF metadata
    convert -size 100x100 xc:red \
        -set "EXIF:GPSLatitude" "37.7749" \
        -set "EXIF:GPSLongitude" "-122.4194" \
        -set "EXIF:DateTime" "2024:12:06 12:00:00" \
        -set "EXIF:Make" "TestCamera" \
        -set "EXIF:Model" "TestModel" \
        sample_exif.jpg
    echo "Created sample_exif.jpg with EXIF data"
else
    echo "ImageMagick not available - tests will use embedded base64 image"
fi
