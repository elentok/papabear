#!/bin/bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
    echo "Usage: $0 <hostname>" >&2
    exit 1
fi

HOST="$1"
BINARY="papabear"
TMPDIR=$(mktemp -d)
REMOTE_SCRIPT="/tmp/papabear-deploy-$$.sh"
trap 'rm -rf "$TMPDIR"' EXIT

echo "Building for linux/amd64..."
GOOS=linux GOARCH=amd64 go build -o "$TMPDIR/$BINARY" .

echo "Uploading to $HOST..."
scp "$TMPDIR/$BINARY" "$HOST:~/$BINARY"

echo
echo "Installing on $HOST..."
ssh "$HOST" "cat > '$REMOTE_SCRIPT' && chmod +x '$REMOTE_SCRIPT'" <<EOF
#!/bin/bash
set -euo pipefail

cleanup() {
    rm -f "$REMOTE_SCRIPT"
}
trap cleanup EXIT

echo "1/3. Installing /usr/local/bin/$BINARY"
sudo install -m 0755 "\$HOME/$BINARY" "/usr/local/bin/$BINARY"
rm "\$HOME/$BINARY"

echo
echo "2/3. Setting up papabear..."
sudo papabear setup

echo
echo "3/3. Restarting papabear..."
sudo systemctl restart papabear
EOF

ssh -t "$HOST" "$REMOTE_SCRIPT"

echo "Done."
echo

echo "Showing Logs..."
ssh -t "$HOST" "papabear logs"
