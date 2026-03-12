#!/usr/bin/env bash
# Deploy clock and/or config services to the Raspberry Pi.
# With no arguments and a TTY: shows an interactive menu.
# With one argument (clock|config|both) or no TTY: runs without menu (for make/CI).
#
# Usage:
#   ./scripts/deploy.sh              # interactive menu (when TTY)
#   ./scripts/deploy.sh both         # deploy both, no menu (e.g. make deploy)
#   ./scripts/deploy.sh clock        # deploy clock only
#   ./scripts/deploy.sh config      # deploy config only
#   ./scripts/deploy.sh --help
#
# Environment: DEPLOY_USER (SSH user on Pi, default: pi), HOST, REMOTE_DIR
# Note: We use DEPLOY_USER not USER so your shell's USER (e.g. jaredwarren) isn't used for SSH.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$REPO_ROOT"

DEPLOY_USER="${DEPLOY_USER:-pi}"
HOST="${HOST:-clock.local}"
REMOTE_DIR="${REMOTE_DIR:-/home/pi/go/github.com/jaredwarren/clock}"
REMOTE="${DEPLOY_USER}@${HOST}"

# Colors and style (no-op if not a TTY or terminal doesn't support)
if [[ -t 1 ]]; then
  BOLD="$(tput bold 2>/dev/null || true)"
  DIM="$(tput dim 2>/dev/null || true)"
  GREEN="$(tput setaf 2 2>/dev/null || true)"
  YELLOW="$(tput setaf 3 2>/dev/null || true)"
  BLUE="$(tput setaf 4 2>/dev/null || true)"
  MAGENTA="$(tput setaf 5 2>/dev/null || true)"
  CYAN="$(tput setaf 6 2>/dev/null || true)"
  RED="$(tput setaf 1 2>/dev/null || true)"
  RESET="$(tput sgr0 2>/dev/null || true)"
else
  BOLD="" DIM="" GREEN="" YELLOW="" BLUE="" MAGENTA="" CYAN="" RED="" RESET=""
fi

usage() {
  echo "Usage: $0 [OPTIONS] [TARGET]"
  echo ""
  echo "With no arguments and a TTY: run interactive menu."
  echo "With TARGET (no menu): deploy and go."
  echo ""
  echo "TARGET:"
  echo "  both    Deploy clockd and configd (default when non-interactive)"
  echo "  clock   Deploy clockd only"
  echo "  config  Deploy configd only"
  echo ""
  echo "Options:"
  echo "  -h, --help  Show this help"
  echo ""
  echo "Environment: DEPLOY_USER (default: pi), HOST, REMOTE_DIR"
  exit 0
}

# Parse help only
for arg in "$@"; do
  if [[ "$arg" == "-h" || "$arg" == "--help" ]]; then
    usage
  fi
done

# Non-interactive or explicit target: set from first arg or default both
DEPLOY_CLOCK=false
DEPLOY_CONFIG=false
DO_BUILD=true
DO_RESTART=true

if [[ $# -ge 1 && "$1" != "-h" && "$1" != "--help" ]]; then
  case "$1" in
    clock)  DEPLOY_CLOCK=true ;;
    config) DEPLOY_CONFIG=true ;;
    both)   DEPLOY_CLOCK=true; DEPLOY_CONFIG=true ;;
    *)      echo "${RED}Unknown target: $1 (use clock, config, or both)${RESET}" >&2; exit 1 ;;
  esac
elif [[ -t 0 ]]; then
  # Interactive menu
  clear
  echo ""
  echo "  ${BOLD}${BLUE}╔══════════════════════════════════════════╗${RESET}"
  echo "  ${BOLD}${BLUE}║${RESET}  ${BOLD}Deploy to ${GREEN}$REMOTE${RESET}${BOLD}${BLUE}                ║${RESET}"
  echo "  ${BOLD}${BLUE}╚══════════════════════════════════════════╝${RESET}"
  echo ""

  # 1) What to deploy
  echo "  ${BOLD}${CYAN}1) What to deploy?${RESET}"
  echo "     ${GREEN}[1]${RESET} Both (clockd + configd)"
  echo "     ${BLUE}[2]${RESET} Clock only (clockd)"
  echo "     ${MAGENTA}[3]${RESET} Config only (configd)"
  echo ""
  read -r -p "  ${DIM}Choose [1-3] (default 1):${RESET} " choice_target
  choice_target="${choice_target:-1}"
  case "$choice_target" in
    1) DEPLOY_CLOCK=true; DEPLOY_CONFIG=true ;;
    2) DEPLOY_CLOCK=true ;;
    3) DEPLOY_CONFIG=true ;;
    *) echo "  ${YELLOW}Invalid choice, using Both.${RESET}"; DEPLOY_CLOCK=true; DEPLOY_CONFIG=true ;;
  esac
  echo ""

  # 2) Build?
  echo "  ${BOLD}${CYAN}2) Build before deploy?${RESET}"
  echo "     ${GREEN}[Y]${RESET} Yes (recommended)"
  echo "     ${DIM}[n] No (use existing binaries)${RESET}"
  echo ""
  read -r -p "  ${DIM}Choose [Y/n]:${RESET} " choice_build
  case "$choice_build" in
    [nN]) DO_BUILD=false ;;
  esac
  echo ""

  # 3) Restart systemd?
  echo "  ${BOLD}${CYAN}3) Restart systemd after rsync?${RESET}"
  echo "     ${GREEN}[Y]${RESET} Yes (recommended)"
  echo "     ${DIM}[n] No${RESET}"
  echo ""
  read -r -p "  ${DIM}Choose [Y/n]:${RESET} " choice_restart
  case "$choice_restart" in
    [nN]) DO_RESTART=false ;;
  esac
  echo ""
else
  # No TTY: default to both, build, restart (for make deploy)
  DEPLOY_CLOCK=true
  DEPLOY_CONFIG=true
fi

echo "  ${BOLD}${BLUE}Summary:${RESET}"
echo "    ${CYAN}Target:${RESET}  $([ "$DEPLOY_CLOCK" = true ] && echo -n "${GREEN}clockd${RESET} ")$([ "$DEPLOY_CONFIG" = true ] && echo -n "${GREEN}configd${RESET}")"
echo "    ${CYAN}Build:${RESET}   ${GREEN}$DO_BUILD${RESET}"
echo "    ${CYAN}Restart:${RESET} ${GREEN}$DO_RESTART${RESET}"
echo ""

# --- Build ---
if [[ "$DO_BUILD" == true ]]; then
  if [[ "$DEPLOY_CLOCK" == true ]]; then
    echo "${YELLOW}>>>${RESET} ${BOLD}${CYAN}Building clockd (ARM)...${RESET}"
    make -C clock build
    echo "    ${GREEN}done.${RESET}"
  fi
  if [[ "$DEPLOY_CONFIG" == true ]]; then
    echo "${YELLOW}>>>${RESET} ${BOLD}${CYAN}Building configd (ARM)...${RESET}"
    GOOS=linux GOARCH=arm GOARM=7 go build -o configd-armv7 -v ./cmd/configd
    echo "    ${GREEN}done.${RESET}"
  fi
fi

# --- Rsync ---
echo "${YELLOW}>>>${RESET} ${BOLD}${CYAN}Rsyncing to ${GREEN}$REMOTE${RESET}${BOLD}${CYAN}...${RESET}"

# Use --progress (portable); macOS rsync doesn't support --info=progress2
RSYNC_OPTS=(-a --progress)

if [[ "$DEPLOY_CLOCK" == true ]]; then
  if [[ -f "$REPO_ROOT/clockd-armv7" ]]; then
    rsync "${RSYNC_OPTS[@]}" "$REPO_ROOT/clockd-armv7" "$REMOTE:$REMOTE_DIR/"
  else
    echo "    ${YELLOW}Warning: clockd-armv7 not found; run with build or build first.${RESET}" >&2
  fi
fi

if [[ "$DEPLOY_CONFIG" == true ]]; then
  if [[ -f "$REPO_ROOT/configd-armv7" ]]; then
    rsync "${RSYNC_OPTS[@]}" "$REPO_ROOT/configd-armv7" "$REMOTE:$REMOTE_DIR/"
  else
    echo "    ${YELLOW}Warning: configd-armv7 not found; run with build or build first.${RESET}" >&2
  fi
  if [[ -d "$REPO_ROOT/templates" ]]; then
    rsync "${RSYNC_OPTS[@]}" "$REPO_ROOT/templates/" "$REMOTE:$REMOTE_DIR/templates/"
  fi
  if [[ -d "$REPO_ROOT/public" ]]; then
    rsync "${RSYNC_OPTS[@]}" "$REPO_ROOT/public/" "$REMOTE:$REMOTE_DIR/public/"
  fi
fi

echo "    ${GREEN}rsync done.${RESET}"

# --- Restart systemd ---
if [[ "$DO_RESTART" == true ]]; then
  echo "${YELLOW}>>>${RESET} ${BOLD}${CYAN}Restarting systemd services...${RESET}"
  RESTART_SERVICES=()
  [[ "$DEPLOY_CLOCK" == true ]]  && RESTART_SERVICES+=(clock.service)
  [[ "$DEPLOY_CONFIG" == true ]] && RESTART_SERVICES+=(config.service)
  ssh "$REMOTE" "sudo systemctl restart ${RESTART_SERVICES[*]}"
  echo "    ${GREEN}restarted: ${RESTART_SERVICES[*]}${RESET}"
fi

echo ""
echo "  ${BOLD}${GREEN}╔══════════════════════════════╗${RESET}"
echo "  ${BOLD}${GREEN}║${RESET}  ${BOLD}Deploy complete.${RESET} ${BOLD}${GREEN}           ║${RESET}"
echo "  ${BOLD}${GREEN}╚══════════════════════════════╝${RESET}"
echo ""
