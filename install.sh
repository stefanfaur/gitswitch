#!/bin/sh
# gitswitch installer — verifies release artifacts with minisign.
# Source: https://github.com/stefanfaur/gitswitch
set -eu

GITSWITCH_REPO="stefanfaur/gitswitch"
# Embedded minisign pubkey. Must match repo-root minisign.pub line 2.
# Enforced by .github/workflows/pubkey-drift.yml.
GITSWITCH_PUBKEY="RWS7seiIU2Cg3+Av3cZoj6QXCGE8mgEoDFsHPCMCmehjaEJBJ904+BPX"

#--- constants ---
DEFAULT_INSTALL_DIR="/usr/local/bin"

#--- fd 3 for tty reads, traps ---
tty_ok=0
if [ -e /dev/tty ] && exec 3</dev/tty 2>/dev/null; then tty_ok=1; fi
if [ "$tty_ok" -eq 0 ] && [ "${ASSUME_YES:-}" != "1" ]; then
  echo "install.sh: no tty available; set ASSUME_YES=1 and env flags" >&2
  exit 1
fi

TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
trap 'printf "\n" >&2; exit 130' INT

#--- helpers ---
die() { echo "install.sh: $*" >&2; exit 1; }

ask() {
  _p="$1"; _d="$2"
  if [ "${ASSUME_YES:-}" = "1" ]; then printf '%s\n' "$_d"; return; fi
  printf '%s [%s]: ' "$_p" "$_d" >&2
  IFS= read -r _ans <&3 || _ans=""
  [ -n "$_ans" ] && printf '%s\n' "$_ans" || printf '%s\n' "$_d"
}

ask_yn() {
  _p="$1"; _d="$2"
  if [ "${ASSUME_YES:-}" = "1" ]; then
    case "$_d" in Y) echo y ;; *) echo n ;; esac; return
  fi
  printf '%s [%s]: ' "$_p" "$_d" >&2
  IFS= read -r _ans <&3 || _ans=""
  if [ -z "$_ans" ]; then
    case "$_d" in Y) echo y ;; *) echo n ;; esac; return
  fi
  case "$_ans" in y|Y|yes|YES) echo y ;; *) echo n ;; esac
}

#--- detect OS / arch ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in linux|darwin) ;; *) die "unsupported OS: $OS" ;; esac
RAW_ARCH=$(uname -m)
case "$RAW_ARCH" in
  x86_64|amd64)  ARCH=amd64 ;;
  aarch64|arm64) ARCH=arm64 ;;
  *) die "unsupported arch: $RAW_ARCH" ;;
esac

#--- sha256 helper ---
if command -v sha256sum >/dev/null 2>&1; then SHASUM="sha256sum"
elif command -v shasum >/dev/null 2>&1; then   SHASUM="shasum -a 256"
else die "need sha256sum or shasum in PATH"; fi

#--- ensure minisign ---
ensure_minisign() {
  command -v minisign >/dev/null 2>&1 && return 0
  PKG_CMD=""
  if   command -v brew     >/dev/null 2>&1; then PKG_CMD="brew install minisign"
  elif command -v apt-get  >/dev/null 2>&1; then PKG_CMD="apt-get install -y minisign"
  elif command -v dnf      >/dev/null 2>&1; then PKG_CMD="dnf install -y minisign"
  elif command -v pacman   >/dev/null 2>&1; then PKG_CMD="pacman -S --noconfirm minisign"
  elif command -v apk      >/dev/null 2>&1; then PKG_CMD="apk add minisign"
  else
    die "minisign not found and no known package manager. Install manually: https://jedisct1.github.io/minisign/"
  fi
  case "$PKG_CMD" in
    brew*) SUDO="" ;;
    *)
      if [ "$(id -u)" -eq 0 ]; then SUDO=""
      elif command -v sudo >/dev/null 2>&1; then SUDO="sudo "
      else die "minisign install needs sudo, which is not installed. Run manually: $PKG_CMD"
      fi
      ;;
  esac
  FULL_CMD="${SUDO}${PKG_CMD}"
  if [ "${GITSWITCH_INSTALL_MINISIGN:-}" = "1" ]; then
    echo ">>> $FULL_CMD" >&2
    sh -c "$FULL_CMD" || die "minisign install failed"
  elif [ "${ASSUME_YES:-}" = "1" ]; then
    die "minisign missing under ASSUME_YES; set GITSWITCH_INSTALL_MINISIGN=1 to auto-accept '$FULL_CMD'"
  else
    ANS=$(ask_yn "minisign not found. Install via '$FULL_CMD'?" "Y")
    [ "$ANS" = "y" ] || die "minisign install declined"
    sh -c "$FULL_CMD" || die "minisign install failed"
  fi
  command -v minisign >/dev/null 2>&1 || die "minisign still not found after install"
}
ensure_minisign

#--- resolve version ---
if [ -n "${GITSWITCH_VERSION:-}" ]; then
  VER="$GITSWITCH_VERSION"
else
  LOC=$(curl -fsI --proto '=https' --tlsv1.2 \
          "https://github.com/${GITSWITCH_REPO}/releases/latest" \
        | awk 'tolower($1)=="location:" {print $2}' | tr -d '\r')
  VER=$(printf '%s' "$LOC" | awk -F/ '{print $NF}')
  [ -n "$VER" ] || die "could not resolve latest version"
fi
case "$VER" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) die "invalid version string: $VER" ;;
esac

#--- detect existing binary, decide upgrade/skip ---
CUR=""
if command -v gitswitch >/dev/null 2>&1; then
  CUR=$(gitswitch --version 2>/dev/null | awk '{print $2}' || true)
fi
if [ -n "$CUR" ]; then
  if [ "$CUR" = "$VER" ]; then
    if [ "${FORCE:-}" = "1" ]; then
      echo "FORCE=1: reinstalling $VER" >&2
    else
      echo "gitswitch $VER already installed; set FORCE=1 to reinstall." >&2
      exit 0
    fi
  elif [ -n "${GITSWITCH_VERSION:-}" ]; then
    echo "switching gitswitch $CUR → $VER (pinned)" >&2
  else
    ANS=$(ask_yn "gitswitch $CUR installed. Upgrade to $VER?" "Y")
    [ "$ANS" = "y" ] || { echo "upgrade declined"; exit 0; }
  fi
fi

#--- resolve install dir ---
if [ -z "${INSTALL_DIR:-}" ]; then
  INSTALL_DIR=$(ask "Install to" "$DEFAULT_INSTALL_DIR")
fi
NEED_SUDO=""
if [ -d "$INSTALL_DIR" ]; then
  [ -w "$INSTALL_DIR" ] || NEED_SUDO="1"
else
  PARENT=$(dirname "$INSTALL_DIR")
  [ -d "$PARENT" ] || die "install dir parent missing: $PARENT"
  [ -w "$PARENT" ] || NEED_SUDO="1"
fi
SUDO_INSTALL=""
if [ -n "$NEED_SUDO" ]; then
  if [ "$(id -u)" -eq 0 ]; then SUDO_INSTALL=""
  elif command -v sudo >/dev/null 2>&1; then SUDO_INSTALL="sudo "
  else die "install dir not writable and sudo not available"; fi
fi

#--- download ---
TARBALL="gitswitch-${VER}-${OS}-${ARCH}.tar.gz"
BASE="https://github.com/${GITSWITCH_REPO}/releases/download/${VER}"
echo "downloading $TARBALL ..." >&2
curl -fsSL --proto '=https' --tlsv1.2 -o "$TMP/$TARBALL"   "$BASE/$TARBALL"   || die "download failed: $BASE/$TARBALL"
curl -fsSL --proto '=https' --tlsv1.2 -o "$TMP/SHA256SUMS" "$BASE/SHA256SUMS" || die "download failed: SHA256SUMS"
curl -fsSL --proto '=https' --tlsv1.2 -o "$TMP/SHA256SUMS.minisig" \
                                                    "$BASE/SHA256SUMS.minisig" || die "download failed: SHA256SUMS.minisig"

#--- verify signature ---
printf 'untrusted comment: gitswitch release pubkey\n%s\n' "$GITSWITCH_PUBKEY" > "$TMP/minisign.pub"
VERIFY_OUT=$(minisign -Vm "$TMP/SHA256SUMS" -p "$TMP/minisign.pub" 2>&1) \
  || { echo "$VERIFY_OUT" >&2; die "minisign verification failed"; }
printf '%s\n' "$VERIFY_OUT" | grep '^Trusted comment:' >&2 || true

#--- verify sha256 ---
( cd "$TMP" && grep "  ${TARBALL}\$" SHA256SUMS | $SHASUM -c - ) \
  || die "sha256 mismatch for $TARBALL"

#--- extract + install ---
( cd "$TMP" && tar -xzf "$TARBALL" )
SRC="$TMP/gitswitch-${VER}-${OS}-${ARCH}/gitswitch"
[ -x "$SRC" ] || die "tarball missing gitswitch binary"
CMD="${SUDO_INSTALL}install -m 0755 \"$SRC\" \"$INSTALL_DIR/gitswitch\""
echo ">>> $CMD" >&2
sh -c "$CMD" || die "install failed"

#--- smoke test ---
"$INSTALL_DIR/gitswitch" --version >/dev/null || die "smoke test failed"
echo "Installed gitswitch $VER to $INSTALL_DIR/gitswitch. Try: gitswitch add <name>" >&2
