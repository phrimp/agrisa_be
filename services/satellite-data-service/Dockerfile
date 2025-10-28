FROM python:3.11 as builder

ENV TZ=Asia/Ho_Chi_Minh \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

WORKDIR /app

# Install build dependencies needed for geospatial packages
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    libgeos-dev \
    libproj-dev \
    gdal-bin \
    libgdal-dev

# Copy only requirements
COPY services/satellite-data-service/requirements.txt .

# Install with cache mount
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --user -r requirements.txt

# ============================================

FROM python:3.11-slim as runtime

ENV TZ=Asia/Ho_Chi_Minh \
    PYTHONUNBUFFERED=1 \
    PATH=/home/app/.local/bin:$PATH

# Install ONLY runtime libraries (no -dev packages)
RUN --mount=type=cache,target=/var/cache/apt,sharing=locked \
    --mount=type=cache,target=/var/lib/apt,sharing=locked \
    apt-get update && \
    # Find the correct libgdal package name
    GDAL_PACKAGE=$(apt-cache search '^libgdal[0-9]+$' | grep -oP 'libgdal\d+' | head -1) && \
    echo "Installing GDAL package: $GDAL_PACKAGE" && \
    apt-get install -y --no-install-recommends \
    tzdata \
    libpq5 \
    libgeos-c1v5 \
    libproj25 \
    gdal-bin \
    $GDAL_PACKAGE \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone

# Create non-root user
RUN useradd --create-home --shell /bin/bash app

WORKDIR /app

# Copy Python packages
COPY --from=builder --chown=app:app /root/.local /home/app/.local

# Copy application code LAST
COPY --chown=app:app services/satellite-data-service/app/ ./app/

USER app

CMD ["python", "-m", "app.main"]
