#!/bin/sh
set -eu

fail() {
	echo "twind: $*" >&2
	exit 1
}

fetch() {
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL -o "$2" "$1"
	else
		wget -qO "$2" "$1"
	fi
}

main() {
	demo=${1:-docs}
	if [ $# -gt 0 ]; then shift; fi
	case $demo in
	docs) set -- docs "$@" ;;
	landing | portfolio | playground | gallery) set -- try "$demo" "$@" ;;
	*) fail "no demo named '$demo': docs, landing, portfolio, playground or gallery" ;;
	esac
	case $(uname -s) in
	Linux) os=linux root=${XDG_CACHE_HOME:-$HOME/.cache}/twind/try ;;
	Darwin) os=darwin root=$HOME/Library/Caches/twind/try ;;
	*) fail "no build for $(uname -s)" ;;
	esac
	case $(uname -m) in
	x86_64 | amd64) arch=amd64 ;;
	aarch64 | arm64) arch=arm64 ;;
	*) fail "no build for $(uname -m)" ;;
	esac
	asset=twind_${os}_$arch.tar.gz
	base=https://github.com/pehcastro/twind/releases/latest/download
	if [ -n "${TWIND_VERSION:-}" ]; then
		base=https://github.com/pehcastro/twind/releases/download/$TWIND_VERSION
	fi
	base=${TWIND_BASE:-$base}

	mkdir -p "$root"
	work=$(mktemp -d "$root/.tmp-XXXXXX")
	trap 'rm -rf "$work"' EXIT
	fetch "$base/checksums.txt" "$work/checksums.txt"
	want=$(awk -v a="$asset" '$2 == a { print $1 }' "$work/checksums.txt")
	case $want in
	*[!0-9a-f]* | '') fail "$base/checksums.txt has no sha256 for $asset" ;;
	esac
	[ ${#want} -eq 64 ] || fail "$base/checksums.txt has no sha256 for $asset"
	dir=$root/$want
	if [ -x "$dir/twind" ]; then
		echo "twind: $asset $want from the cache in $dir" >&2
	else
		echo "twind: fetching $base/$asset" >&2
		fetch "$base/$asset" "$work/$asset"
		if command -v sha256sum >/dev/null 2>&1; then
			got=$(sha256sum "$work/$asset" | cut -d' ' -f1)
		else
			got=$(shasum -a 256 "$work/$asset" | cut -d' ' -f1)
		fi
		[ "$got" = "$want" ] || fail "$asset has sha256 $got, checksums.txt says $want: refusing to run it"
		echo "twind: sha256 $got matches checksums.txt" >&2
		mkdir "$work/twind"
		tar -xzf "$work/$asset" -C "$work/twind"
		mv "$work/twind" "$dir"
	fi
	rm -rf "$work"
	trap - EXIT

	if [ ! -t 0 ] && (: </dev/tty) 2>/dev/null; then
		"$dir/twind" "$@" </dev/tty
	else
		"$dir/twind" "$@"
	fi
}

main "$@"
