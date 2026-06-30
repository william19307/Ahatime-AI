#!/usr/bin/env bash
# Deploy ahatime-new-api from an uploaded source package or directory under /root.
#
# Typical use on the server:
#   bash /root/deploy-ahatime.sh
#
# Optional explicit source:
#   bash /root/deploy-ahatime.sh /root/ahatime-new-api-main.tar.gz
#   bash /root/deploy-ahatime.sh /root/ahatime-new-api
#
# Environment overrides:
#   APP_DIR=/opt/ahatime UPLOAD_ROOT=/root SERVICE=newapi bash /root/deploy-ahatime.sh

set -Eeuo pipefail

APP_DIR="${APP_DIR:-/opt/ahatime}"
SRC_DIR="${SRC_DIR:-$APP_DIR/src}"
UPLOAD_ROOT="${UPLOAD_ROOT:-/root}"
BACKUP_DIR="${BACKUP_DIR:-$APP_DIR/backups}"
SERVICE="${SERVICE:-newapi}"
CONTAINER="${CONTAINER:-ahatime-gateway}"
MYSQL_CONTAINER="${MYSQL_CONTAINER:-ahatime-mysql}"
STATUS_URL="${STATUS_URL:-http://127.0.0.1:3000/api/status}"
KEEP_BACKUPS="${KEEP_BACKUPS:-10}"
NO_CACHE="${NO_CACHE:-1}"

TS="$(date +%F_%H%M%S)"
TMP_DIR=""

log() {
  printf '\n==> %s\n' "$*"
}

die() {
  printf '\n!! %s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [ -n "$TMP_DIR" ] && [ -d "$TMP_DIR" ]; then
    rm -rf "$TMP_DIR"
  fi
}
trap cleanup EXIT

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令: $1"
}

validate_source_dir() {
  local dir="$1"
  [ -d "$dir" ] || return 1
  [ -f "$dir/go.mod" ] || return 1
  [ -d "$dir/controller" ] || return 1
  [ -d "$dir/web" ] || return 1
}

find_source_dir_inside() {
  local root="$1"
  local gomod
  gomod="$(find "$root" -maxdepth 5 -type f -name go.mod -print -quit)"
  [ -n "$gomod" ] || return 1
  dirname "$gomod"
}

unpack_archive() {
  local archive="$1"
  TMP_DIR="$(mktemp -d /tmp/ahatime-src.XXXXXX)"

  case "$archive" in
    *.tar.gz|*.tgz)
      tar -xzf "$archive" -C "$TMP_DIR"
      ;;
    *.tar)
      tar -xf "$archive" -C "$TMP_DIR"
      ;;
    *.zip)
      require_command unzip
      unzip -q "$archive" -d "$TMP_DIR"
      ;;
    *)
      die "不支持的源码包格式: $archive"
      ;;
  esac

  find_source_dir_inside "$TMP_DIR"
}

find_latest_upload() {
  local latest

  latest="$(find "$UPLOAD_ROOT" -maxdepth 2 -type f \
    \( -name '*.tar.gz' -o -name '*.tgz' -o -name '*.tar' -o -name '*.zip' \) \
    -printf '%T@ %p\n' 2>/dev/null | sort -nr | awk 'NR==1{print substr($0, index($0,$2))}')"
  if [ -n "$latest" ]; then
    printf '%s\n' "$latest"
    return 0
  fi

  latest="$(find "$UPLOAD_ROOT" -maxdepth 2 -type f -name go.mod \
    -printf '%T@ %h\n' 2>/dev/null | sort -nr | awk 'NR==1{print substr($0, index($0,$2))}')"
  [ -n "$latest" ] || return 1
  printf '%s\n' "$latest"
}

resolve_source() {
  local input="${1:-}"
  local source_path

  if [ -n "$input" ]; then
    source_path="$input"
  else
    source_path="$(find_latest_upload)" || die "在 $UPLOAD_ROOT 下没找到源码包或源码目录"
  fi

  if [ -d "$source_path" ]; then
    if validate_source_dir "$source_path"; then
      printf '%s\n' "$source_path"
      return 0
    fi
    local nested
    nested="$(find_source_dir_inside "$source_path")" || die "目录里没找到 go.mod: $source_path"
    validate_source_dir "$nested" || die "源码目录不完整: $nested"
    printf '%s\n' "$nested"
    return 0
  fi

  [ -f "$source_path" ] || die "源码路径不存在: $source_path"
  local unpacked
  unpacked="$(unpack_archive "$source_path")" || die "源码包里没找到 go.mod: $source_path"
  validate_source_dir "$unpacked" || die "源码包目录不完整: $unpacked"
  printf '%s\n' "$unpacked"
}

backup_configs() {
  local config_backup="$BACKUP_DIR/config_$TS.tgz"
  local files=()

  [ -f "$APP_DIR/docker-compose.yml" ] && files+=("docker-compose.yml")
  [ -f "$APP_DIR/.env" ] && files+=(".env")
  [ -f "$APP_DIR/Caddyfile" ] && files+=("Caddyfile")

  if [ "${#files[@]}" -gt 0 ]; then
    tar -czf "$config_backup" -C "$APP_DIR" "${files[@]}"
    echo "配置备份: $config_backup"
  fi
}

backup_database() {
  local db_backup="$BACKUP_DIR/newapi_$TS.sql"

  if ! docker ps --format '{{.Names}}' | grep -qx "$MYSQL_CONTAINER"; then
    echo "跳过数据库备份: 未找到运行中的 $MYSQL_CONTAINER"
    return 0
  fi

  if docker exec "$MYSQL_CONTAINER" sh -c 'command -v mysqldump >/dev/null 2>&1'; then
    if docker exec "$MYSQL_CONTAINER" sh -c 'mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" --single-transaction --quick newapi' > "$db_backup"; then
      echo "数据库备份: $db_backup"
      return 0
    fi
  fi

  rm -f "$db_backup"
  echo "警告: 数据库备份失败，继续部署前请确认是否可接受"
}

backup_source() {
  local source_backup="$BACKUP_DIR/src_$TS.tgz"

  [ -d "$SRC_DIR" ] || die "生产源码目录不存在: $SRC_DIR"
  tar -czf "$source_backup" -C "$APP_DIR" "$(basename "$SRC_DIR")"
  echo "源码备份: $source_backup"
}

sync_source() {
  local source_dir="$1"

  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete \
      --exclude '.git' \
      --exclude 'output' \
      --exclude 'node_modules' \
      --exclude 'web/default/node_modules' \
      --exclude 'web/classic/node_modules' \
      "$source_dir"/ "$SRC_DIR"/
  else
    echo "未安装 rsync，使用 tar 管道同步"
    rm -rf "$SRC_DIR"
    mkdir -p "$SRC_DIR"
    tar -C "$source_dir" \
      --exclude '.git' \
      --exclude 'output' \
      --exclude 'node_modules' \
      --exclude 'web/default/node_modules' \
      --exclude 'web/classic/node_modules' \
      -cf - . | tar -C "$SRC_DIR" -xf -
  fi
}

prune_backups() {
  local pattern="$1"
  local keep="$2"
  find "$BACKUP_DIR" -maxdepth 1 -type f -name "$pattern" -printf '%T@ %p\n' 2>/dev/null \
    | sort -nr \
    | awk -v keep="$keep" 'NR>keep {print $2}' \
    | xargs -r rm -f
}

verify_service() {
  docker compose ps
  docker logs --tail=100 "$CONTAINER" || true

  for _ in $(seq 1 20); do
    if curl -fsS "$STATUS_URL" >/dev/null; then
      echo "健康检查通过: $STATUS_URL"
      return 0
    fi
    sleep 3
  done

  die "健康检查失败: $STATUS_URL。请查看 docker logs --tail=200 $CONTAINER"
}

main() {
  require_command docker
  require_command tar
  require_command find
  require_command awk
  require_command curl

  [ -d "$APP_DIR" ] || die "应用目录不存在: $APP_DIR"
  [ -f "$APP_DIR/docker-compose.yml" ] || die "没找到 compose 文件: $APP_DIR/docker-compose.yml"
  mkdir -p "$BACKUP_DIR"

  local source_dir
  source_dir="$(resolve_source "${1:-}")"
  log "使用源码: $source_dir"

  log "备份当前生产环境"
  backup_source
  backup_configs
  backup_database

  log "同步源码到 $SRC_DIR"
  sync_source "$source_dir"

  log "重建并启动 $SERVICE"
  cd "$APP_DIR"
  if [ "$NO_CACHE" = "1" ]; then
    docker compose build --no-cache "$SERVICE"
  else
    docker compose build "$SERVICE"
  fi
  docker compose up -d "$SERVICE"

  log "验证服务"
  verify_service

  log "清理旧备份（保留最近 $KEEP_BACKUPS 份）"
  prune_backups 'src_*.tgz' "$KEEP_BACKUPS"
  prune_backups 'config_*.tgz' "$KEEP_BACKUPS"
  prune_backups 'newapi_*.sql' "$KEEP_BACKUPS"

  cat <<EOF

部署完成。

如需回滚源码：
  cd $APP_DIR
  rm -rf src
  tar -xzf $BACKUP_DIR/src_$TS.tgz -C $APP_DIR
  docker compose build --no-cache $SERVICE
  docker compose up -d $SERVICE
EOF
}

main "$@"
