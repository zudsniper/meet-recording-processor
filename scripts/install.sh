#!/usr/bin/env bash
set -euo pipefail

# Cute installer for Meet Recording Processor (mrp)
# Usage (interactive):   curl -fsSL https://raw.githubusercontent.com/zudsniper/meet-recording-processor/main/scripts/install.sh | bash
# Usage (non-interactive): curl -fsSL ... | bash -s -- -y

YES="false"
if [[ ${1:-} == "-y" || ${1:-} == "--yes" || ${1:-} == "--non-interactive" ]]; then
  YES="true"
fi

emoji_info="[\033[34mℹ\033[0m]"
emoji_ok="[\033[32m✔\033[0m]"
emoji_warn="[\033[33m⚠\033[0m]"
emoji_err="[\033[31m✖\033[0m]"

spinner() {
  local pid=$1 msg=$2
  local frames=("🌑" "🌒" "🌓" "🌔" "🌕" "🌖" "🌗" "🌘")
  local i=0
  tput civis || true
  while kill -0 "$pid" 2>/dev/null; do
    printf "\r%s %s %s" "$emoji_info" "${frames[$i]}" "$msg"
    i=$(((i+1)%${#frames[@]}))
    sleep 0.12
  done
  printf "\r"
  tput cnorm || true
}

run() {
  local msg="$1"; shift
  local log
  log=$(mktemp)
  ("$@" >"$log" 2>&1) &
  local pid=$!
  spinner "$pid" "$msg"
  wait "$pid" && { printf "%s %s\n" "$emoji_ok" "$msg"; rm -f "$log"; return 0; } || {
    printf "%s %s\n" "$emoji_err" "$msg"
    printf "\n$emoji_err Command failed. Logs:\n"
    sed 's/^/  /' "$log" || true
    rm -f "$log"
    return 1
  }
}

say() { printf "%b %s\n" "$emoji_info" "$*"; }
warn() { printf "%b %s\n" "$emoji_warn" "$*"; }
ok() { printf "%b %s\n" "$emoji_ok" "$*"; }
err() { printf "%b %s\n" "$emoji_err" "$*"; }

banner() {
cat << 'EOF'
   __  __ _____ _____          ____                 _           _           
  |  \/  | ____| ____|  🗣️   |  _ \ ___  ___ _ __ | |__   ___ | |_ ___ _ __ 
  | |\/| |  _| |  _|    🎥   | |_) / _ \/ _ \ '_ \| '_ \ / _ \| __/ _ \ '__|
  | |  | | |___| |___   ✍️   |  _ <  __/  __/ |_) | | | | (_) | ||  __/ |   
  |_|  |_|_____|_____|  🧠   |_| \_\___|\___| .__/|_| |_|\___/ \__\___|_|   
                                            |_|                              
EOF
}

require_sudo() {
  if [[ $EUID -ne 0 ]]; then
    if command -v sudo >/dev/null 2>&1; then
      SUDO="sudo"
    else
      SUDO=""
      warn "sudo not found; system-wide installs may fail. Falling back to user directories."
    fi
  else
    SUDO=""
  fi
}

detect_os() {
  UNAME_S=$(uname -s)
  case "$UNAME_S" in
    Darwin) OS="mac" ;;
    Linux)  OS="linux" ;;
    *)      OS="linux" ;; # Treat others as linux-ish
  esac
  if [[ -f /etc/os-release ]]; then
    # shellcheck disable=SC1091
    . /etc/os-release
    DISTRO_ID=${ID:-}
    DISTRO_VERSION_ID=${VERSION_ID:-}
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) ARCH="amd64" ;;
  esac
}

pkg_manager=""
detect_pkg_manager() {
  for pm in apt-get dnf yum pacman zypper; do
    if command -v "$pm" >/dev/null 2>&1; then pkg_manager=$pm; return; fi
  done
  pkg_manager=""
}

install_brew() {
  if ! command -v brew >/dev/null 2>&1; then
    if [[ "$YES" == "false" ]]; then
      read -r -p "Install Homebrew? [Y/n] " ans; ans=${ans:-Y}
      [[ $ans =~ ^[Yy]$ ]] || { warn "Skipping Homebrew"; return; }
    fi
    run "Installing Homebrew" /bin/bash -c "$(/usr/bin/curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    # Add brew to PATH for current session
    if [[ -d /opt/homebrew/bin ]]; then eval "$(/opt/homebrew/bin/brew shellenv)"; fi
    if [[ -d /usr/local/bin ]]; then eval "$(/usr/local/bin/brew shellenv)"; fi
  fi
}

install_ffmpeg_mac() { run "Installing ffmpeg (brew)" brew install ffmpeg; }
install_python_mac() { run "Installing Python 3 (brew)" brew install python; }
install_go_mac() { run "Installing Go (brew)" brew install go; }

install_ffmpeg_linux() {
  case "$pkg_manager" in
    apt-get) run "Updating apt" $SUDO env DEBIAN_FRONTEND=noninteractive apt-get update && run "Installing ffmpeg" $SUDO env DEBIAN_FRONTEND=noninteractive apt-get install -y ffmpeg ;;
    dnf)     run "Installing ffmpeg" $SUDO dnf install -y ffmpeg || run "Enabling RPM Fusion" $SUDO dnf install -y https://download1.rpmfusion.org/free/fedora/rpmfusion-free-release-$(rpm -E %fedora).noarch.rpm && run "Installing ffmpeg" $SUDO dnf install -y ffmpeg ;;
    yum)     run "Installing ffmpeg" $SUDO yum install -y epel-release && run "Installing ffmpeg" $SUDO yum install -y ffmpeg ;;
    pacman)  run "Installing ffmpeg" $SUDO pacman -Sy --noconfirm ffmpeg ;;
    zypper)  run "Installing ffmpeg" $SUDO zypper --non-interactive install ffmpeg ;;
    *)       warn "Unknown package manager; skipping ffmpeg install" ;;
  esac
}

install_python_linux() {
  case "$pkg_manager" in
    apt-get) run "Updating apt" $SUDO env DEBIAN_FRONTEND=noninteractive apt-get update && run "Installing Python3+pip+venv" $SUDO env DEBIAN_FRONTEND=noninteractive apt-get install -y python3 python3-pip python3-venv ;;
    dnf)     run "Installing Python3+pip+venv" $SUDO dnf install -y python3 python3-pip python3-virtualenv || true ;;
    yum)     run "Installing Python3+pip+venv" $SUDO yum install -y python3 python3-pip python3-virtualenv || true ;;
    pacman)  run "Installing Python3+pip" $SUDO pacman -Sy --noconfirm python python-pip ;;
    zypper)  run "Installing Python3+pip" $SUDO zypper --non-interactive install python3 python3-pip ;;
    *)       warn "Unknown package manager; ensure Python3 and pip are installed" ;;
  esac
}

setup_venv() {
  local venv_dir="$HOME/.mrp/venv"
  mkdir -p "$HOME/.mrp"
  if [[ ! -d "$venv_dir" ]]; then
    run "Creating Python venv" python3 -m venv "$venv_dir"
  fi
  # Upgrade pip and install faster-whisper in venv
  run "Upgrading pip in venv" "$venv_dir/bin/python" -m pip install --upgrade pip
  run "Installing faster-whisper" "$venv_dir/bin/python" -m pip install faster-whisper
  export MRP_PY="$venv_dir/bin/python"
}

install_go_linux() {
  local GOV=1.22.0
  local OSSTR=linux
  local TAR=go${GOV}.${OSSTR}-${ARCH}.tar.gz
  local URL=https://go.dev/dl/${TAR}
  run "Downloading Go ${GOV}" bash -lc "curl -fsSL ${URL} -o /tmp/${TAR}"
  if [[ -d /usr/local/go ]]; then run "Removing old Go" $SUDO rm -rf /usr/local/go; fi
  run "Installing Go to /usr/local" $SUDO tar -C /usr/local -xzf /tmp/${TAR}
  rm -f /tmp/${TAR}
}

ensure_go_in_path() {
  if ! command -v go >/dev/null 2>&1; then
    export PATH="/usr/local/go/bin:$PATH"
  fi
}

build_and_install_mrp() {
  local target_dir
  if [[ -w /usr/local/bin ]] || [[ -n "${SUDO}" ]]; then
    target_dir=/usr/local/bin
  else
    target_dir="$HOME/.local/bin"
    mkdir -p "$target_dir"
  fi
  run "Building mrp" bash -lc "GO111MODULE=on go build -o mrp ./cmd/mrp"
  if [[ -n "${SUDO}" && "$target_dir" == "/usr/local/bin" ]]; then
    run "Installing mrp to ${target_dir}" $SUDO mv mrp ${target_dir}/mrp
  else
    run "Installing mrp to ${target_dir}" mv mrp ${target_dir}/mrp
  fi
  case ":$PATH:" in
    *":$target_dir:"*) ;;
    *) warn "${target_dir} not in PATH; consider adding it." ;;
  esac
}

setup_env() {
  if [[ "$YES" == "false" ]]; then
    echo
    read -r -p "Add API keys now? [y/N] " do_keys; do_keys=${do_keys:-N}
    if [[ $do_keys =~ ^[Yy]$ ]]; then
      read -r -p "OpenAI API key (leave blank to skip): " OPENAI_API_KEY || true
      read -r -p "Cloudflare Account ID (blank to skip): " CF_ACCOUNT_ID || true
      read -r -p "Cloudflare API Token (blank to skip): " CF_API_TOKEN || true
      local envfile="$HOME/.mrp.env"
      {
        [[ -n "${OPENAI_API_KEY:-}" ]] && echo "export OPENAI_API_KEY='$OPENAI_API_KEY'"
        [[ -n "${CF_ACCOUNT_ID:-}" ]] && echo "export CF_ACCOUNT_ID='$CF_ACCOUNT_ID'"
        [[ -n "${CF_API_TOKEN:-}" ]] && echo "export CF_API_TOKEN='$CF_API_TOKEN'"
        [[ -n "${MRP_PY:-}" ]] && echo "export MRP_PY='$MRP_PY'"
      } >> "$envfile"
      ok "Saved keys to $envfile"

      local shell_rc
      if [[ -n "${ZSH_VERSION:-}" ]]; then shell_rc="$HOME/.zshrc"; else shell_rc="$HOME/.bashrc"; fi
      if ! grep -q ".mrp.env" "$shell_rc" 2>/dev/null; then
        echo "source \"$envfile\"" >> "$shell_rc"
        ok "Appended source line to $shell_rc"
      fi
    fi
  fi
}

main() {
  banner
  say "Welcome! Let’s get you set up 🚀"
  require_sudo
  detect_os
  detect_arch
  if [[ "$OS" == "linux" ]]; then detect_pkg_manager; fi

  say "Checking and installing prerequisites 🧩"
  if [[ "$OS" == "mac" ]]; then
    install_brew || true
    command -v ffmpeg >/dev/null 2>&1 || install_ffmpeg_mac || true
    command -v python3 >/dev/null 2>&1 || install_python_mac || true
    command -v go >/dev/null 2>&1 || install_go_mac || true
  else
    command -v ffmpeg >/dev/null 2>&1 || install_ffmpeg_linux || true
    command -v python3 >/dev/null 2>&1 || install_python_linux || true
    command -v go >/dev/null 2>&1 || install_go_linux || true
  fi
  ensure_go_in_path
  setup_venv || warn "Could not set up venv; faster-whisper may be missing"

  say "Building and installing mrp 🛠️"
  build_and_install_mrp

  setup_env

  ok "Installation complete! ✨ Try: \033[1m mrp --help \033[0m"
  say "Local backend uses faster-whisper in a venv."
  say "Ubuntu ${DISTRO_VERSION_ID:-} detected: using apt-get where applicable."
}

main "$@"
