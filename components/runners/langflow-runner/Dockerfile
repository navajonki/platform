FROM python:3.11-slim

WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir \
    requests==2.31.0 \
    kubernetes==28.1.0

# Copy runner script
COPY langflow-runner/run.py /app/run.py
RUN chmod +x /app/run.py

# Run as non-root user
RUN useradd -m -u 1000 runner && chown -R runner:runner /app
USER 1000

ENTRYPOINT ["python3", "/app/run.py"]
