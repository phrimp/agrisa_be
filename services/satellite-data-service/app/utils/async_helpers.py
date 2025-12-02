"""
Async helpers for running blocking Google Earth Engine operations in thread pool.

This module provides utilities to run synchronous GEE API calls in a thread pool,
preventing them from blocking the asyncio event loop and enabling true parallel processing.
"""

import asyncio
import logging
from concurrent.futures import ThreadPoolExecutor
from functools import partial
from typing import Any, Callable, TypeVar

logger = logging.getLogger(__name__)

# Global thread pool executor for blocking GEE operations
# Max workers: 10 (balance between parallelism and GEE API rate limits)
_executor: ThreadPoolExecutor | None = None
_executor_max_workers = 10

# Semaphore to limit concurrent GEE API calls (prevent rate limit issues)
_gee_semaphore: asyncio.Semaphore | None = None
_gee_max_concurrent = 15

T = TypeVar("T")


def get_executor() -> ThreadPoolExecutor:
    """
    Get or create the global thread pool executor.

    Returns:
        ThreadPoolExecutor: Shared executor for blocking operations
    """
    global _executor
    if _executor is None:
        _executor = ThreadPoolExecutor(
            max_workers=_executor_max_workers,
            thread_name_prefix="gee_worker_"
        )
        logger.info(f"Created thread pool executor with {_executor_max_workers} workers")
    return _executor


def get_semaphore() -> asyncio.Semaphore:
    """
    Get or create the global GEE API semaphore for rate limiting.

    Returns:
        asyncio.Semaphore: Semaphore limiting concurrent GEE calls
    """
    global _gee_semaphore
    if _gee_semaphore is None:
        _gee_semaphore = asyncio.Semaphore(_gee_max_concurrent)
        logger.info(f"Created GEE semaphore with {_gee_max_concurrent} concurrent limit")
    return _gee_semaphore


async def run_in_executor(func: Callable[..., T], *args: Any, **kwargs: Any) -> T:
    """
    Run a blocking function in the thread pool executor.

    This prevents blocking the asyncio event loop and allows other requests
    to be processed while waiting for the blocking operation to complete.

    Args:
        func: Blocking function to execute
        *args: Positional arguments for the function
        **kwargs: Keyword arguments for the function

    Returns:
        Result of the blocking function

    Example:
        >>> result = await run_in_executor(blocking_gee_call, param1, param2=value)
    """
    loop = asyncio.get_event_loop()
    executor = get_executor()

    # Use partial to bind kwargs if provided
    if kwargs:
        func_with_args = partial(func, *args, **kwargs)
        return await loop.run_in_executor(executor, func_with_args)
    else:
        return await loop.run_in_executor(executor, func, *args)


async def run_in_executor_with_limit(
    func: Callable[..., T], *args: Any, **kwargs: Any
) -> T:
    """
    Run a blocking function in the thread pool with GEE API rate limiting.

    Uses a semaphore to limit concurrent GEE API calls, preventing rate limit errors
    while still allowing parallel processing within limits.

    Args:
        func: Blocking function to execute
        *args: Positional arguments for the function
        **kwargs: Keyword arguments for the function

    Returns:
        Result of the blocking function

    Example:
        >>> result = await run_in_executor_with_limit(gee_api_call, coordinates)
    """
    semaphore = get_semaphore()
    async with semaphore:
        return await run_in_executor(func, *args, **kwargs)


async def gather_with_limit(
    *coros: Any,
    limit: int | None = None,
    return_exceptions: bool = False
) -> list[Any]:
    """
    Run multiple coroutines concurrently with optional limit on parallelism.

    Args:
        *coros: Coroutines to execute
        limit: Optional maximum concurrent operations (None = unlimited)
        return_exceptions: If True, exceptions are returned instead of raised

    Returns:
        List of results from all coroutines

    Example:
        >>> tasks = [process_image(img) for img in images]
        >>> results = await gather_with_limit(*tasks, limit=5)
    """
    if limit is None or limit >= len(coros):
        # No limit or limit >= tasks, run all in parallel
        return await asyncio.gather(*coros, return_exceptions=return_exceptions)

    # Chunk coroutines and process in batches
    results = []
    for i in range(0, len(coros), limit):
        batch = coros[i : i + limit]
        batch_results = await asyncio.gather(*batch, return_exceptions=return_exceptions)
        results.extend(batch_results)

    return results


def shutdown_executor():
    """
    Shutdown the global thread pool executor.

    Should be called during application shutdown to clean up resources.
    """
    global _executor
    if _executor is not None:
        logger.info("Shutting down thread pool executor")
        _executor.shutdown(wait=True)
        _executor = None
