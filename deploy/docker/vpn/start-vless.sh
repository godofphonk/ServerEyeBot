#!/bin/bash

# VLESS/Shadowsocks startup script for ServerEyeBot

echo "Starting ServerEyeBot with VLESS VPN support..."

# Check if VLESS configuration exists
if [ -f "/app/config/vless-config.json" ]; then
    echo "VLESS configuration found, starting VPN..."
    
    # Extract VLESS URL from config
    VLESS_URL=$(python3 -c "
import json
with open('/app/config/vless-config.json', 'r') as f:
    config = json.load(f)
    print(config.get('vless', ''))
")
    
    if [ ! -z "$VLESS_URL" ]; then
        echo "Starting VLESS connection..."
        
        # Create v2ray config
        cat > /tmp/v2ray-config.json << EOF
{
  "log": {
    "loglevel": "info"
  },
  "inbounds": [
    {
      "port": 1080,
      "protocol": "socks",
      "settings": {
        "auth": "noauth"
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "$(python3 -c "
import json
with open('/app/config/vless-config.json', 'r') as f:
    config = json.load(f)
    print(config.get('server', ''))
")",
            "port": $(python3 -c "
import json
with open('/app/config/vless-config.json', 'r') as f:
    config = json.load(f)
    print(config.get('port', 443))
"),
            "users": [
              {
                "id": "$(python3 -c "
import json
with open('/app/config/vless-config.json', 'r') as f:
    config = json.load(f)
    print(config.get('uuid', ''))
")",
                "encryption": "none"
              }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "ws",
        "security": "tls",
        "wsSettings": {
          "path": "$(python3 -c "
import json
with open('/app/config/vless-config.json', 'r') as f:
    config = json.load(f)
    print(config.get('path', '/'))
")",
          "headers": {
            "Host": "$(python3 -c "
import json
with open('/app/config/vless-config.json', 'r') as f:
    config = json.load(f)
    print(config.get('host', ''))
")"
          }
        }
      }
    }
  ]
}
EOF
        
        # Start v2ray in background
        v2ray run -config /tmp/v2ray-config.json &
        V2RAY_PID=$!
        
        # Wait for VPN to connect
        echo "Waiting for VLESS to establish connection..."
        sleep 10
        
        # Configure proxy for system
        export http_proxy=socks5://127.0.0.1:1080
        export https_proxy=socks5://127.0.0.1:1080
        
        echo "VLESS is running with PID: $V2RAY_PID"
        
        # Test connectivity
        echo "Testing VPN connectivity..."
        timeout 10 curl -s --socks5 127.0.0.1:1080 http://ifconfig.me || echo "VPN test failed"
        
    else
        echo "Invalid VLESS configuration"
    fi
else
    echo "No VLESS configuration found, running without VPN"
fi

# Start the bot
echo "Starting ServerEyeBot..."
exec ./servereye-bot
