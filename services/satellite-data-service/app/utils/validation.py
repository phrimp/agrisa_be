import re
from typing import Tuple, Optional
from app.config.settings import get_settings

settings = get_settings()


def validate_coordinates(latitude: float, longitude: float) -> Tuple[bool, Optional[str]]:
    """
    Validate latitude and longitude coordinates.
    
    Args:
        latitude: Latitude value
        longitude: Longitude value
        
    Returns:
        Tuple of (is_valid, error_message)
    """
    # Check basic range validation
    if not (-90 <= latitude <= 90):
        return False, f"Latitude must be between -90 and 90, got {latitude}"
    
    if not (-180 <= longitude <= 180):
        return False, f"Longitude must be between -180 and 180, got {longitude}"
    
    return True, None


def validate_vietnam_bounds(latitude: float, longitude: float) -> Tuple[bool, Optional[str]]:
    """
    Validate if coordinates are within Vietnam boundaries.
    
    Args:
        latitude: Latitude value
        longitude: Longitude value
        
    Returns:
        Tuple of (within_vietnam, warning_message)
    """
    bounds = settings.vietnam_bounds
    
    within_vietnam = (
        bounds["south"] <= latitude <= bounds["north"] and
        bounds["west"] <= longitude <= bounds["east"]
    )
    
    if not within_vietnam:
        return False, "Coordinates are outside Vietnam boundaries"
    
    return True, None


def validate_date_format(date_string: str) -> Tuple[bool, Optional[str]]:
    """
    Validate date string format (YYYY-MM-DD).
    
    Args:
        date_string: Date string to validate
        
    Returns:
        Tuple of (is_valid, error_message)
    """
    pattern = r'^\d{4}-\d{2}-\d{2}$'
    
    if not re.match(pattern, date_string):
        return False, "Date must be in YYYY-MM-DD format"
    
    try:
        from datetime import datetime
        datetime.strptime(date_string, '%Y-%m-%d')
        return True, None
    except ValueError as e:
        return False, f"Invalid date: {str(e)}"


def validate_dimensions(dimensions: str) -> Tuple[bool, Optional[str]]:
    """
    Validate image dimensions string format (e.g., "512x512").
    
    Args:
        dimensions: Dimensions string to validate
        
    Returns:
        Tuple of (is_valid, error_message)
    """
    try:
        if 'x' not in dimensions:
            return False, "Dimensions must be in format 'WIDTHxHEIGHT'"
        
        width_str, height_str = dimensions.split('x')
        width, height = int(width_str), int(height_str)
        
        if width <= 0 or height <= 0:
            return False, "Width and height must be positive integers"
        
        if width > 2048 or height > 2048:
            return False, "Maximum dimensions are 2048x2048 pixels"
        
        if width < 64 or height < 64:
            return False, "Minimum dimensions are 64x64 pixels"
        
        return True, None
        
    except ValueError:
        return False, "Dimensions must be integers in format 'WIDTHxHEIGHT'"


def validate_scale(scale: int) -> Tuple[bool, Optional[str]]:
    """
    Validate image scale (meters per pixel).
    
    Args:
        scale: Scale value in meters per pixel
        
    Returns:
        Tuple of (is_valid, error_message)
    """
    if scale < 10:
        return False, "Scale must be at least 10 meters per pixel"
    
    if scale > 1000:
        return False, "Scale cannot exceed 1000 meters per pixel"
    
    return True, None
