#!/bin/sh
set -a  # автоматически экспортировать все переменные
[ -f .env ] && . ./.env
set +a
cd server && go run ./cmd/api "$@"
