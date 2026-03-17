#!/usr/bin/env bash

export AUTH_HEADER

login() {
  local user="$1"
  echo "$user"
}

function helper {
  local tmp="ok"
  printf "helper\n"
}
