#!/usr/bin/env bash

export AUTH_HEADER
source ./lib/common.sh

login() {
  local user="$1"
  echo "$user"
}

function helper {
  local tmp="ok"
  printf "helper\n"
}
