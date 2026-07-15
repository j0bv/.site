#!/bin/bash
# Deployment script for truck-inventory service

set -e

SERVICE_NAME="truck-inventory"
INSTALL_DIR="/opt/truck-inventory"
SERVICE_USER="truck_service"

echo "🚛 Deploying Truck Inventory Service..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ Please run as root (sudo)"
    exit 1
fi

# Create service user if it doesn't exist
if ! id "$SERVICE_USER" &>/dev/null; then
    echo "📝 Creating service user: $SERVICE_USER"
    useradd -r -s /bin/false -d "$INSTALL_DIR" "$SERVICE_USER"
fi

# Create installation directory
echo "📁 Creating installation directory..."
mkdir -p "$INSTALL_DIR"
chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"

# Copy files
echo "📦 Copying files..."
cp truck-inventory "$INSTALL_DIR/"
cp .env "$INSTALL_DIR/" 2>/dev/null || echo "⚠️  Warning: .env file not found. Please create it manually."
chown "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"/*

# Install systemd service
echo "⚙️  Installing systemd service..."
cp deployment/truck-inventory.service /etc/systemd/system/
systemctl daemon-reload

# Enable and start service
echo "🚀 Starting service..."
systemctl enable "$SERVICE_NAME"
systemctl start "$SERVICE_NAME"

# Check status
sleep 2
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "✅ Service started successfully!"
    echo ""
    echo "📊 Service status:"
    systemctl status "$SERVICE_NAME" --no-pager -l
    echo ""
    echo "📋 View logs: journalctl -u $SERVICE_NAME -f"
else
    echo "❌ Service failed to start. Check logs: journalctl -u $SERVICE_NAME"
    exit 1
fi
