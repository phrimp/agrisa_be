import secrets
import string
from datetime import datetime, timezone
from typing import Optional


# Character set for random string generation
CHARSET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789"


def generate_random_string(length: int = 8) -> str:
    """
    Generate a cryptographically secure random string of specified length.

    Args:
        length: Length of the random string to generate

    Returns:
        Random string using alphanumeric characters
    """
    return "".join(secrets.choice(CHARSET) for _ in range(length))


def generate_model_id(prefix: str, random_length: int = 8) -> str:
    """
    Generate a model ID with prefix and random suffix.

    Args:
        prefix: 2-letter prefix (e.g., 'ST', 'EX', 'AC')
        random_length: Length of random suffix

    Returns:
        Formatted ID string (e.g., 'STAb3dE2k9')

    Raises:
        ValueError: If prefix is not exactly 2 characters
    """
    if len(prefix) != 2:
        raise ValueError("Prefix must be exactly 2 characters")

    if not prefix.isalpha():
        raise ValueError("Prefix must contain only alphabetic characters")

    prefix = prefix.upper()
    random_suffix = generate_random_string(random_length)

    return f"{prefix}{random_suffix}"


def get_current_timestamp() -> int:
    """
    Get current timestamp in seconds since Unix epoch.

    Returns:
        Current timestamp as integer seconds
    """
    return int(datetime.now(timezone.utc).timestamp())


def timestamp_to_datetime(timestamp: int) -> datetime:
    """
    Convert timestamp (seconds) to UTC datetime object.

    Args:
        timestamp: Unix timestamp in seconds

    Returns:
        UTC datetime object
    """
    return datetime.fromtimestamp(timestamp, tz=timezone.utc)


def datetime_to_timestamp(dt: datetime) -> int:
    """
    Convert datetime to timestamp (seconds).

    Args:
        dt: Datetime object

    Returns:
        Unix timestamp in seconds
    """
    return int(dt.timestamp())


# Model-specific ID generators
def generate_satellite_image_id() -> str:
    """Generate ID for SatelliteImage model (SI_xxxxxxxx)."""
    return generate_model_id("SI")


def generate_image_analysis_id() -> str:
    """Generate ID for ImageAnalysis model (IA_xxxxxxxx)."""
    return generate_model_id("IA")


def generate_satellite_collection_id() -> str:
    """Generate ID for SatelliteCollection model (SC_xxxxxxxx)."""
    return generate_model_id("SC")


def generate_download_request_id() -> str:
    """Generate ID for DownloadRequest model (DR_xxxxxxxx)."""
    return generate_model_id("DR")


# Validation utilities
def is_valid_model_id(model_id: str, expected_prefix: Optional[str] = None) -> bool:
    """
    Validate model ID format.

    Args:
        model_id: ID to validate
        expected_prefix: Optional expected prefix (e.g., 'SI')

    Returns:
        True if ID format is valid
    """
    if not isinstance(model_id, str):
        return False

    parts = model_id.split("_")
    if len(parts) != 2:
        return False

    prefix, suffix = parts

    # Validate prefix
    if len(prefix) != 2 or not prefix.isalpha() or not prefix.isupper():
        return False

    if expected_prefix and prefix != expected_prefix:
        return False

    # Validate suffix (8 alphanumeric characters)
    if len(suffix) != 8:
        return False

    return all(c in CHARSET for c in suffix)


def extract_prefix_from_id(model_id: str) -> Optional[str]:
    """
    Extract prefix from model ID.

    Args:
        model_id: ID to extract prefix from

    Returns:
        Prefix string or None if invalid format
    """
    if not is_valid_model_id(model_id):
        return None

    return model_id.split("_")[0]
