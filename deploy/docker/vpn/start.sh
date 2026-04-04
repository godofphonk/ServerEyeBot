#!/bin/sh

# VPN startup script for ServerEyeBot

echo "Starting ServerEyeBot with VPN support..."

# Check if VPN configuration exists
if [ -f "/app/config/vpn.conf" ]; then
    echo "VPN configuration found, starting VPN..."
    
    # Start OpenVPN in background
    openvpn --config /app/config/vpn.conf --daemon --writepid /app/vpn/vpn.pid
    
    # Wait for VPN to connect
    echo "Waiting for VPN to establish connection..."
    sleep 10
    
    # Check if VPN is connected
    if [ -f "/app/vpn/vpn.pid" ]; then
        echo "VPN is running with PID: $(cat /app/vpn/vpn.pid)"
        
        # Wait a bit more for full connection
        sleep 5
        
        # Check network connectivity through VPN
        echo "Testing VPN connectivity..."
        
        # Show IP address
        IP=$(wget -qO- http://ifconfig.me || echo "Unknown")
        echo "Current IP address: $IP"
    else
        echo "Failed to start VPN"
    fi
else
    echo "No VPN configuration found, running without VPN"
fi

# Start the bot
echo "Starting ServerEyeBot..."
exec ./servereye-bot
