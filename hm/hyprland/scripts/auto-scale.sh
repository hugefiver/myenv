#!/usr/bin/env bash
set -euo pipefail

MIN_RES=2560
MIN_DIAG_INCH=24
TARGET_SCALE="1.25"

name="" res_w=0 res_h=0 phys_w=0 phys_h=0

apply_if_needed() {
  [[ -z "$name" || "$phys_w" -eq 0 ]] && return 0

  local res_max=$(( res_w > res_h ? res_w : res_h ))
  (( res_max < MIN_RES )) && return 0

  local diag
  diag=$(awk "BEGIN { printf \"%d\", sqrt(${phys_w}^2 + ${phys_h}^2) / 25.4 }")
  (( diag < MIN_DIAG_INCH )) && return 0

  hyprctl keyword monitor "$name,preferred,auto,$TARGET_SCALE" >/dev/null 2>&1
}

while IFS= read -r line; do
  if [[ "$line" =~ ^([A-Za-z0-9_-]+)\ .* ]]; then
    apply_if_needed
    name="${BASH_REMATCH[1]}"
    res_w=0; res_h=0; phys_w=0; phys_h=0
  fi
  if [[ "$line" =~ "Physical size: "([0-9]+)x([0-9]+) ]]; then
    phys_w="${BASH_REMATCH[1]}"
    phys_h="${BASH_REMATCH[2]}"
  fi
  if [[ "$line" =~ ([0-9]+)x([0-9]+)" px,".*current ]]; then
    res_w="${BASH_REMATCH[1]}"
    res_h="${BASH_REMATCH[2]}"
  fi
done < <(wlr-randr 2>/dev/null)

apply_if_needed
