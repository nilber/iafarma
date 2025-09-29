# Runtime image with FFmpeg support
FROM alpine:latest

# Install runtime dependencies including FFmpeg
RUN apk --no-cache add \
    ca-certificates \
    ffmpeg \
    tzdata \
    wget \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy the compiled binary from GitHub Actions build
COPY zoe-connet-api .

# Copy CSV data files
COPY municipios_com_uf.csv .

# Create directories for file processing
RUN mkdir -p /app/temp /app/uploads /app/logs && \
    chown -R appuser:appgroup /app && \
    chmod +x /app/zoe-connet-api

# Switch to non-root user
USER appuser

# Expose port 8080
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./zoe-connet-api"]
