FROM python:3.11-slim as builder

# Set timezone
ENV TZ=Asia/Ho_Chi_Minh

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    libpq-dev \
    libgeos-dev \
    libproj-dev \
    libgdal-dev \
    build-essential \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy and install Python dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir --user -r requirements.txt

FROM python:3.11-slim as runtime

# Set timezone to Asia/Ho_Chi_Minh
ENV TZ=Asia/Ho_Chi_Minh

# Install only runtime dependencies
RUN apt-get update && apt-get install -y \
    tzdata \
    libpq5 \
    libgeos-c1v5 \
    libproj25 \
    libgdal32 \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd --create-home --shell /bin/bash app

# Set working directory
WORKDIR /app

# Copy Python packages from builder stage
COPY --from=builder /root/.local /home/app/.local

# Copy application code
COPY --chown=app:app app/ ./app/

# Switch to non-root user
USER app

# Update PATH to include user packages
ENV PATH=/home/app/.local/bin:$PATH

# Run the application
CMD ["python", "-m", "app.main"]
