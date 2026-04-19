# --- Builder stage: install Python dependencies ---
FROM python:3.13-alpine AS builder

WORKDIR /build

COPY requirements.txt .
RUN pip install --no-cache-dir --prefix=/install -r requirements.txt

# --- Runtime stage: minimal image with app ---
FROM python:3.13-alpine

WORKDIR /app

# Copy installed dependencies from builder
COPY --from=builder /install /usr/local

# Copy application source
COPY app.py .

# Non-root user for runtime
RUN adduser -D -u 10001 appuser
USER 10001

EXPOSE 8080

CMD ["gunicorn", "--bind", "0.0.0.0:8080", "--workers", "2", "--access-logfile", "-", "app:app"]
