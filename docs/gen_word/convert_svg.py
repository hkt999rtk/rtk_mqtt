#!/usr/bin/env python3
"""
Convert SVG logo to PNG with transparent background
"""

from wand.image import Image as WandImage
from PIL import Image

def convert_svg_to_png():
    # Read SVG and convert to PNG
    with WandImage(filename='realtek-logo.svg', background='transparent') as img:
        # Resize to appropriate size for header
        img.resize(150, 75)
        
        # Convert to blob
        png_blob = img.make_blob(format='png')
    
    # Open with PIL to ensure proper format
    with open('page-logo.png', 'wb') as f:
        f.write(png_blob)
    
    print('SVG converted to PNG successfully')

if __name__ == "__main__":
    convert_svg_to_png()