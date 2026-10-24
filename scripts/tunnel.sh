#!/usr/bin/env bash
# Secure Payment Gateway - Public Tunnel Launcher for Remote Interviews & Demos
set -e

PORT=${1:-8080}

echo -e "\033[1;36m==========================================================\033[0m"
echo -e "\033[1;33m  SECURE PAYMENT GATEWAY - PUBLIC TUNNEL LAUNCHER         \033[0m"
echo -e "\033[1;36m==========================================================\033[0m"
echo -e "Target Local Service : http://localhost:$PORT\n"

if command -v cloudflared &> /dev/null; then
    echo -e "\033[1;32m[STARTING] Launching Cloudflare Tunnel on port $PORT...\033[0m"
    echo -e "Cloudflare will output a public HTTPS link (e.g. https://*.trycloudflare.com):\n"
    cloudflared tunnel --url "http://localhost:$PORT"
elif command -v ngrok &> /dev/null; then
    echo -e "\033[1;32m[STARTING] Launching ngrok on port $PORT...\033[0m"
    ngrok http "$PORT"
else
    echo -e "\033[1;31m[ERROR] Neither 'cloudflared' nor 'ngrok' was found!\033[0m"
    echo "Install Cloudflare Tunnel: brew install cloudflared (macOS) or apt install cloudflared (Debian/Ubuntu)"
    echo "Or install ngrok: https://ngrok.com/download"
    exit 1
fi

